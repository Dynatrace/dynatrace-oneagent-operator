package kubesystem

import "os"

const (
	deployedOnOpenshift = "DEPLOYED_ON_OPENSHIFT"
)

type KubeSystem struct {
	IsDeployedOnOpenshift bool
}

func NewKubeSystem() *KubeSystem {
	return &KubeSystem{
		IsDeployedOnOpenshift: os.Getenv(deployedOnOpenshift) == "true",
	}
}
