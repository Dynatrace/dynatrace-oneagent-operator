package v1alpha1

func SetDefaults_OneAgentSpec(obj *OneAgentSpec) {
	if obj.WaitReadySeconds == nil {
		obj.WaitReadySeconds = new(uint16)
		*obj.WaitReadySeconds = 300
	}
}
