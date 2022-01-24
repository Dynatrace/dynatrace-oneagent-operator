package oneagent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/kubesystem"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

const (
	testImage = "registry/repository/image:tag"
)

func TestPreparePodSpecInstaller(t *testing.T) {
	t.Run(`default docker image is set if instance spec is not set`, func(t *testing.T) {
		instance := &v1alpha1.OneAgent{}
		podSpec := &v1.PodSpec{
			Containers: []v1.Container{{}},
		}
		daemonSet := daemonSetBuilder{
			instance:   instance,
			kubeSystem: &kubesystem.KubeSystem{IsDeployedViaOLM: false},
		}
		err := daemonSet.preparePodSpecInstaller(podSpec)

		assert.NoError(t, err)
		assert.Equal(t, oneagentDockerImage, podSpec.Containers[0].Image)
	})
	t.Run(`image is set to the redhat registry image if deployed via OLM`, func(t *testing.T) {
		instance := &v1alpha1.OneAgent{}
		podSpec := &v1.PodSpec{
			Containers: []v1.Container{{}},
		}
		daemonSet := daemonSetBuilder{
			instance:   instance,
			kubeSystem: &kubesystem.KubeSystem{IsDeployedViaOLM: true},
		}
		err := daemonSet.preparePodSpecInstaller(podSpec)

		assert.NoError(t, err)
		assert.Equal(t, oneagentRedhatImage, podSpec.Containers[0].Image)
	})
	t.Run(`image is set to instance spec if instance spec is set`, func(t *testing.T) {
		instance := &v1alpha1.OneAgent{
			Spec: v1alpha1.OneAgentSpec{
				Image: testImage,
			},
		}
		podSpec := &v1.PodSpec{
			Containers: []v1.Container{{}},
		}
		daemonSet := daemonSetBuilder{
			instance:   instance,
			kubeSystem: &kubesystem.KubeSystem{IsDeployedViaOLM: false},
		}
		err := daemonSet.preparePodSpecInstaller(podSpec)

		assert.NoError(t, err)
		assert.Equal(t, testImage, podSpec.Containers[0].Image)

		daemonSet.kubeSystem.IsDeployedViaOLM = true
		err = daemonSet.preparePodSpecInstaller(podSpec)

		assert.NoError(t, err)
		assert.Equal(t, testImage, podSpec.Containers[0].Image)
	})
}
