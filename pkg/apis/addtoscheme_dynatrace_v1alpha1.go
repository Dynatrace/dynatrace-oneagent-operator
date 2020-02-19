package apis

import (
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme, istiov1alpha3.SchemeBuilder.AddToScheme)
}
