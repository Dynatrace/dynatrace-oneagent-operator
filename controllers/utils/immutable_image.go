package utils

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
)

// SetUseImmutableImageStatus updates the status' UseImmutableImage field to indicate whether the Operator should use
// immutable images or not.
func SetUseImmutableImageStatus(instance dynatracev1alpha1.BaseOneAgent) bool {
	if instance.GetSpec().UseImmutableImage == instance.GetStatus().UseImmutableImage {
		return false
	}

	instance.GetStatus().UseImmutableImage = instance.GetSpec().UseImmutableImage
	return true
}
