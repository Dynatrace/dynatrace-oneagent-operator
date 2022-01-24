package kubesystem

import "os"

type KubeSystem struct {
	IsDeployedViaOLM bool
}

func NewKubeSystem() *KubeSystem {
	return &KubeSystem{
		IsDeployedViaOLM: os.Getenv("DEPLOYED_VIA_OLM") == "true",
	}
}
