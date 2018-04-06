package v1alpha1

func SetDefaults_OneAgent(obj *OneAgent) {
	if obj.Spec.WaitReadySeconds == nil {
		obj.Spec.WaitReadySeconds = new(int32)
		*obj.Spec.WaitReadySeconds = 300
	}
}
