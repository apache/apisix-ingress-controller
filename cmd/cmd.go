package cmd

import (
	"github.com/spf13/cobra"

	"github.com/api7/ingress-controller/cmd/ingress"
)

// NewAPISIXIngressControllerCommand creates the apisix-ingress-controller command.
func NewAPISIXIngressControllerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apisix-ingress-controller [command]",
		Short:   "Yet another Ingress controller for Kubernetes using Apache APISIX as the high performance reverse proxy.",
		Version: "", // TODO: fill the version info.
	}

	cmd.AddCommand(ingress.NewIngressCommand())
	return cmd
}
