package annotations

import (
	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_rewriteTarget              = "k8s.apisix.apache.org/rewrite-target"
	_rewriteTargetRegex         = "k8s.apisix.apache.org/rewrite-target-regex"
	_rewriteTargetRegexTemplate = "k8s.apisix.apache.org/rewrite-target-regex-template"
)

type rewrite struct{}

// NewIPRestrictionHandler creates a handler to convert
// annotations about client ips control to APISIX ip-restrict plugin.
func NewRewriteHandler() Handler {
	return &rewrite{}
}

func (i *rewrite) PluginName() string {
	return "proxy-rewrite"
}

func (i *rewrite) Handle(e Extractor) (interface{}, error) {
	var plugin apisixv1.RewriteConfig
	log.Errorw("handle rewrite annotations")
	rewriteTarget := e.GetStringAnnotation(_rewriteTarget)
	rewriteTargetRegex := e.GetStringAnnotation(_rewriteTargetRegex)
	rewriteTemplate := e.GetStringAnnotation(_rewriteTargetRegexTemplate)
	if rewriteTarget != "" || rewriteTargetRegex != "" || rewriteTemplate != "" {
		plugin.RewriteTarget = rewriteTarget
		if rewriteTargetRegex != "" && rewriteTemplate != "" {
			plugin.RewriteTargetRegex = []string{rewriteTargetRegex, rewriteTemplate}
		}
		return &plugin, nil
	}
	return nil, nil
}
