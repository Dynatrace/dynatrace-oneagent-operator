package apis

import (
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	istiov1alpha3 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/istio/v1alpha3"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes.Register(v1alpha1.RegisterDefaults)
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme, istiov1alpha3.SchemeBuilder.AddToScheme)
}
