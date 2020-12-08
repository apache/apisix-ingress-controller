package main

import (
	"os"

	"github.com/api7/ingress-controller/cmd"
	"github.com/api7/ingress-controller/log"
)

func main(){
	root := cmd.NewAPISIXIngressControllerCommand()
	if err := root.Execute(); err != nil {
		log.GetLogger().Error(err.Error())
		os.Exit(1)
	}
}
