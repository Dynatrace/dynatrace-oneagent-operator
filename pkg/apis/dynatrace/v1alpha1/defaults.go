package v1alpha1

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

func SetDefaults_OneAgentSpec(obj *OneAgentSpec) {
	if obj.WaitReadySeconds == nil {
		obj.WaitReadySeconds = new(uint16)
		*obj.WaitReadySeconds = 300
	}

	if obj.Image == "" {
		obj.Image = "docker.io/dynatrace/oneagent:latest"
	}

	if _, ok := obj.NodeSelector["beta.kubernetes.io/os"]; !ok {
		obj.NodeSelector["beta.kubernetes.io/os"] = "linux"
	}

	// temporary map for easy lookup of entries in obj.Env
	env := make(map[string]int)
	for i, e := range obj.Env {
		env[e.Name] = i
	}
	if _, ok := env["ONEAGENT_INSTALLER_SCRIPT_URL"]; !ok {
		obj.Env = append(obj.Env, corev1.EnvVar{
			Name:  "ONEAGENT_INSTALLER_SCRIPT_URL",
			Value: fmt.Sprintf("%s/v1/deployment/installer/agent/unix/default/latest?Api-Token=%s&arch=x86&flavor=default", obj.ApiUrl, "$(ONEAGENT_INSTALLER_TOKEN)"),
		})
	}
	if i, ok := env["ONEAGENT_INSTALLER_SKIP_CERT_CHECK"]; !ok {
		obj.Env = append(obj.Env, corev1.EnvVar{
			Name:  "ONEAGENT_INSTALLER_SKIP_CERT_CHECK",
			Value: strconv.FormatBool(obj.SkipCertCheck),
		})
	} else {
		obj.Env[i].Value = strconv.FormatBool(obj.SkipCertCheck)
	}
}
