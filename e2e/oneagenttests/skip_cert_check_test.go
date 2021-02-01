package oneagenttests

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestSkipCertCheck(t *testing.T) {
	apiURL, clt := prepareDefaultEnvironment(t)
	oneAgent := createMinimumViableOneAgent(apiURL)

	oneAgent.Spec.SkipCertCheck = true

	_ = deployOneAgent(clt, &oneAgent, t)
	_, podList := findOneAgentPods(clt, t)

	assert.NotEmpty(t, podList.Items)

	for _, pod := range podList.Items {
		assert.NotEmpty(t, pod.Spec.Containers)
		checkContainer(t, pod.Spec.Containers)
	}
}

func checkContainer(t *testing.T, containers []corev1.Container) {
	for _, container := range containers {
		assert.NotEmpty(t, container.Env)
		checkEnvVars(t, container.Env)
	}
}

func checkEnvVars(t *testing.T, envs []corev1.EnvVar) {
	for _, env := range envs {
		if env.Name == keySkipCertCheck {
			enabled, err := strconv.ParseBool(env.Value)
			assert.NoError(t, err)
			assert.True(t, enabled)
			break
		}
	}
}
