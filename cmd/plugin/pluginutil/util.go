package pluginutil

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func GetNamespaces(flags *genericclioptions.ConfigFlags) string {
	ns, _, err := flags.ToRawKubeConfigLoader().Namespace()
	if err != nil || len(ns) == 0 {
		ns = apiv1.NamespaceDefault
		return ns
	}

	return ns
}

func GetKubeconfigFile(flags *genericclioptions.ConfigFlags) string {
	if *flags.KubeConfig == "" {
		return "~/.kube/conf"
	}
	return *flags.KubeConfig
}
