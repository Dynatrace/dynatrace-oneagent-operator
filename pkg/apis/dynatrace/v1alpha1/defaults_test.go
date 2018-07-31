package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetDefaults_OneAgentSpec(t *testing.T) {
	oa := newOneAgentSpec()
	SetDefaults_OneAgentSpec(oa)
	assert.NotNil(t, oa.WaitReadySeconds)
	assert.NotEmpty(t, oa.Image)
	assert.NotEmpty(t, oa.NodeSelector)
	assert.NotEmpty(t, oa.Env)
}

func newOneAgentSpec() *OneAgentSpec {
	return &OneAgentSpec{}
}
