package apisix

import (
	"strconv"
	apisix "github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
	seven "github.com/gxthrj/seven/apisix"
)

// BuildAnnotation return plugins and group
func BuildAnnotation(annotations map[string]string) (apisix.Plugins, string){
	plugins := make(apisix.Plugins)
	cors := &CorsYaml{}
	// ingress.class
	group := ""
	for k, v := range annotations {
		switch {
		case k == SSLREDIRECT:
			if b, err := strconv.ParseBool(v); err == nil && b {
				// todo add ssl-redirect plugin
			}
		case k == WHITELIST:
			ipRestriction := seven.BuildIpRestriction(&v, nil)
			plugins["ip-restriction"] = ipRestriction
		case k == ENABLE_CORS:
			cors.SetEnable(v)
		case k == CORS_ALLOW_ORIGIN:
			cors.SetOrigin(v)
		case k == CORS_ALLOW_HEADERS:
			cors.SetHeaders(v)
		case k == CORS_ALLOW_METHODS:
			cors.SetMethods(v)
		case k == INGRESS_CLASS:
			group = v
		default:
			// do nothing
		}
	}
	// build CORS plugin
	if cors.Enable {
		plugins["aispeech-cors"] = cors.Build()
	}
	return plugins, group
}
