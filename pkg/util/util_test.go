package util

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildPodList(t *testing.T) {
	desired := &corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
	pList := BuildPodList()
	assert.True(t, reflect.DeepEqual(desired, pList))
}

func TestBuildLabels(t *testing.T) {
	l := BuildLabels("my-name")
	assert.Equal(t, l["dynatrace"], "oneagent")
	assert.Equal(t, l["oneagent"], "my-name")
}

func TestBuildDaemonSet(t *testing.T) {
	ds := BuildDaemonSet("my-name", "my-namespace")
	assert.Equal(t, ds.APIVersion, "apps/v1")
	assert.Equal(t, ds.Kind, "DaemonSet")
	assert.Equal(t, ds.Name, "my-name")
	assert.Equal(t, ds.Namespace, "my-namespace")
}

func TestBuildSecret(t *testing.T) {
	s := BuildSecret("my-name", "my-namespace")
	assert.Equal(t, s.Name, "my-name")
	assert.Equal(t, s.Namespace, "my-namespace")
}

func TestGetPodReadyState(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{},
		}}
	assert.True(t, GetPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}}
	assert.True(t, GetPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: false}}
	assert.False(t, GetPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}, {Ready: true}}
	assert.True(t, GetPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}, {Ready: false}}
	assert.False(t, GetPodReadyState(pod))
}
