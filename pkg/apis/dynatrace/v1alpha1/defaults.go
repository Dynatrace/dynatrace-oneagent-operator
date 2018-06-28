package v1alpha1

func SetDefaults_OneAgentSpec(obj *OneAgentSpec) {
	if obj.WaitReadySeconds == nil {
		obj.WaitReadySeconds = new(uint16)
		*obj.WaitReadySeconds = 300
	}

	if obj.Image == "" {
		obj.Image = "docker.io/dynatrace/oneagent:latest"
	}

	if len(obj.Args) == 0 {
		obj.Args = append(obj.Args, "APP_LOG_CONTENT_ACCESS=1")
	}
}
