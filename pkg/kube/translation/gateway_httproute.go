package translation

import (
	"fmt"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	"strings"
)

func (t *translator) TranslateGatewayHTTPRouteV1Alpha2(httpRoute *gatewayv1alpha2.HTTPRoute) (*TranslateContext, error) {
	ctx := defaultEmptyTranslateContext()

	var hosts []string
	for _, hostname := range httpRoute.Spec.Hostnames {
		hosts = append(hosts, string(hostname))
	}

	rules := httpRoute.Spec.Rules

	for i, rule := range rules {
		backends := rule.BackendRefs
		if len(backends) == 0 {
			continue
		}

		matches := rule.Matches
		if len(matches) == 0 {
			defaultType := gatewayv1alpha2.PathMatchPathPrefix
			defaultValue := "/"
			matches = []gatewayv1alpha2.HTTPRouteMatch{
				{
					Path: &gatewayv1alpha2.HTTPPathMatch{
						Type:  &defaultType,
						Value: &defaultValue,
					},
				},
			}
		}

		for j, match := range matches {
			route, err := t.translateGatewayHTTPRouteMatch(&match)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to translate Rules[%v].Matches[%v]", i, j))
			}

			route.Hosts = hosts

			ctx.addRoute(route)
		}

		//TODO: Support filters
		//filters := rule.Filters

		for j, backend := range backends {
			//TODO: Support filters
			//filters := backend.Filters
			kind := strings.ToLower(string(*backend.Kind))
			if kind != "service" {
				log.Warnw(fmt.Sprintf("ignore non-service kind at Rules[%v].BackendRefs[%v]", i, j),
					zap.String("kind", kind),
				)
				continue
			}

			ns := string(*backend.Namespace)
			if ns != httpRoute.Namespace {
				// TODO: check gatewayv1alpha2.ReferencePolicy
			}

			ups, err := t.TranslateUpstream(ns, string(backend.Name), "", int32(*backend.Port))
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to translate Rules[%v].BackendRefs[%v]", i, j))
			}
			ctx.addUpstream(ups)
		}
	}

	return ctx, nil
}

func (t *translator) translateGatewayHTTPRouteMatch(match *gatewayv1alpha2.HTTPRouteMatch) (*apisixv1.Route, error) {
	route := apisixv1.NewDefaultRoute()

	if match.Path != nil {
		switch *match.Path.Type {
		case gatewayv1alpha2.PathMatchExact:
			route.Uri = *match.Path.Value
		case gatewayv1alpha2.PathMatchPathPrefix:
			route.Uri = *match.Path.Value + "*"
		case gatewayv1alpha2.PathMatchRegularExpression:
			var this []apisixv1.StringOrSlice
			this = append(this, apisixv1.StringOrSlice{
				StrVal: "uri",
			})
			this = append(this, apisixv1.StringOrSlice{
				StrVal: "~~",
			})
			this = append(this, apisixv1.StringOrSlice{
				StrVal: *match.Path.Value,
			})

			route.Vars = append(route.Vars, this)
		default:
			return nil, errors.New("unknown path match type " + string(*match.Path.Type))
		}
	}

	if match.Headers != nil && len(match.Headers) > 0 {
		for _, header := range match.Headers {
			name := strings.ToLower(string(header.Name))
			name = strings.ReplaceAll(name, "-", "_")

			var this []apisixv1.StringOrSlice
			this = append(this, apisixv1.StringOrSlice{
				StrVal: "http_" + name,
			})

			switch *header.Type {
			case gatewayv1alpha2.HeaderMatchExact:
				this = append(this, apisixv1.StringOrSlice{
					StrVal: "==",
				})
			case gatewayv1alpha2.HeaderMatchRegularExpression:
				this = append(this, apisixv1.StringOrSlice{
					StrVal: "~~",
				})
			default:
				return nil, errors.New("unknown header match type " + string(*header.Type))
			}

			this = append(this, apisixv1.StringOrSlice{
				StrVal: *match.Path.Value,
			})

			route.Vars = append(route.Vars, this)
		}
	}

	if match.QueryParams != nil && len(match.QueryParams) > 0 {
		for _, query := range match.QueryParams {
			var this []apisixv1.StringOrSlice
			this = append(this, apisixv1.StringOrSlice{
				StrVal: "arg_" + strings.ToLower(query.Name),
			})

			switch *query.Type {
			case gatewayv1alpha2.QueryParamMatchExact:
				this = append(this, apisixv1.StringOrSlice{
					StrVal: "==",
				})
			case gatewayv1alpha2.QueryParamMatchRegularExpression:
				this = append(this, apisixv1.StringOrSlice{
					StrVal: "~~",
				})
			default:
				return nil, errors.New("unknown query match type " + string(*query.Type))
			}

			this = append(this, apisixv1.StringOrSlice{
				StrVal: *match.Path.Value,
			})

			route.Vars = append(route.Vars, this)
		}
	}

	if match.Method != nil {
		route.Methods = []string{
			string(*match.Method),
		}
	}

	return route, nil
}
