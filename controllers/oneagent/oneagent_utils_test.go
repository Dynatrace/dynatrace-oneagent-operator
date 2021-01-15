package oneagent

import (
	"errors"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/dtclient"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildLabels(t *testing.T) {
	l := buildLabels("my-name")
	assert.Equal(t, l["dynatrace"], "oneagent")
	assert.Equal(t, l["oneagent"], "my-name")
}

func TestGetPodReadyState(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{},
		}}
	assert.True(t, getPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}}
	assert.True(t, getPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: false}}
	assert.False(t, getPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}, {Ready: true}}
	assert.True(t, getPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}, {Ready: false}}
	assert.False(t, getPodReadyState(pod))
}

func TestOneAgent_Validate(t *testing.T) {
	oa := newOneAgent()
	assert.Error(t, validate(oa))
	oa.Spec.APIURL = "https://f.q.d.n/api"
	assert.NoError(t, validate(oa))
}

func TestMigrationForDaemonSetWithoutAnnotation(t *testing.T) {
	oaKey := metav1.ObjectMeta{Name: "my-oneagent", Namespace: "my-namespace"}

	ds1 := &appsv1.DaemonSet{ObjectMeta: oaKey}

	ds2, err := newDaemonSetForCR(consoleLogger, &dynatracev1alpha1.OneAgent{ObjectMeta: oaKey}, "cluster1")
	assert.NoError(t, err)
	assert.NotEmpty(t, ds2.Annotations[annotationTemplateHash])

	assert.True(t, hasDaemonSetChanged(ds1, ds2))
}

func TestHasSpecChanged(t *testing.T) {
	runTest := func(msg string, exp bool, mod func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent)) {
		t.Run(msg, func(t *testing.T) {
			key := metav1.ObjectMeta{Name: "my-oneagent", Namespace: "my-namespace"}
			old := dynatracev1alpha1.OneAgent{ObjectMeta: key}
			new := dynatracev1alpha1.OneAgent{ObjectMeta: key}

			mod(&old, &new)

			ds1, err := newDaemonSetForCR(consoleLogger, &old, "cluster1")
			assert.NoError(t, err)

			ds2, err := newDaemonSetForCR(consoleLogger, &new, "cluster1")
			assert.NoError(t, err)

			assert.NotEmpty(t, ds1.Annotations[annotationTemplateHash])
			assert.NotEmpty(t, ds2.Annotations[annotationTemplateHash])

			assert.Equal(t, exp, hasDaemonSetChanged(ds1, ds2))
		})
	}

	runTest("no changes", false, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {})

	runTest("image added", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		new.Spec.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image set but no change", false, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		old.Spec.Image = "docker.io/dynatrace/oneagent"
		new.Spec.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image removed", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		old.Spec.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image changed", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		old.Spec.Image = "registry.access.redhat.com/dynatrace/oneagent"
		new.Spec.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("argument removed", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		old.Spec.Args = []string{"INFRA_ONLY=1", "--set-host-property=OperatorVersion=snapshot"}
		new.Spec.Args = []string{"INFRA_ONLY=1"}
	})

	runTest("argument changed", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		old.Spec.Args = []string{"INFRA_ONLY=1"}
		new.Spec.Args = []string{"INFRA_ONLY=0"}
	})

	runTest("all arguments removed", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		old.Spec.Args = []string{"INFRA_ONLY=1"}
	})

	runTest("resources added", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		new.Spec.Resources = newResourceRequirements()
	})

	runTest("resources removed", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		old.Spec.Resources = newResourceRequirements()
	})

	runTest("resources removed", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		old.Spec.Resources = newResourceRequirements()
	})

	runTest("priority class added", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		new.Spec.PriorityClassName = "class"
	})

	runTest("priority class removed", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		old.Spec.PriorityClassName = "class"
	})

	runTest("priority class set but no change", false, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		old.Spec.PriorityClassName = "class"
		new.Spec.PriorityClassName = "class"
	})

	runTest("priority class changed", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		old.Spec.PriorityClassName = "some class"
		new.Spec.PriorityClassName = "other class"
	})

	runTest("dns policy added", true, func(old *dynatracev1alpha1.OneAgent, new *dynatracev1alpha1.OneAgent) {
		new.Spec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
	})
}

func TestGetPodsToRestart(t *testing.T) {
	dtc := new(dtclient.MockDynatraceClient)
	dtc.On("GetAgentVersionForIP", "127.0.0.1").Return("1.2.3", nil)
	dtc.On("GetAgentVersionForIP", "127.0.0.2").Return("0.1.2", nil)
	dtc.On("GetAgentVersionForIP", "127.0.0.3").Return("", errors.New("n/a"))

	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-1"},
			Spec:       corev1.PodSpec{NodeName: "node-1"},
			Status:     corev1.PodStatus{HostIP: "127.0.0.1"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-2"},
			Spec:       corev1.PodSpec{NodeName: "node-2"},
			Status:     corev1.PodStatus{HostIP: "127.0.0.2"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-3"},
			Spec:       corev1.PodSpec{NodeName: "node-3"},
			Status:     corev1.PodStatus{HostIP: "127.0.0.3"},
		},
	}
	oa := newOneAgent()
	oa.Status.Version = "1.2.3"
	oa.Status.Instances = map[string]dynatracev1alpha1.OneAgentInstance{"node-3": {Version: "outdated"}}
	doomed, err := findOutdatedPodsInstaller(pods, dtc, oa, consoleLogger)
	assert.Lenf(t, doomed, 1, "list of pods to restart")
	assert.Equalf(t, doomed[0], pods[1], "list of pods to restart")
	assert.Equal(t, nil, err)
}

func newOneAgent() *dynatracev1alpha1.OneAgent {
	return &dynatracev1alpha1.OneAgent{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OneAgent",
			APIVersion: "dynatrace.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-oneagent",
			Namespace: "my-namespace",
			UID:       "69e98f18-805a-42de-84b5-3eae66534f75",
		},
	}
}

func newResourceRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    parseQuantity("10m"),
			"memory": parseQuantity("100Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    parseQuantity("20m"),
			"memory": parseQuantity("200Mi"),
		},
	}
}

func parseQuantity(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}
