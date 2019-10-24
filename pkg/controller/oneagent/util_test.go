package oneagent

import (
	"errors"
	"reflect"
	"testing"

	api "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
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
	oa.Spec.ApiUrl = "https://f.q.d.n/api"
	assert.NoError(t, validate(oa))
}

func TestHasSpecChanged(t *testing.T) {
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		assert.Falsef(t, hasSpecChanged(ds, oa), "empty specs change detected")
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Image: "docker.io/dynatrace/oneagent",
		}}
		oa := newOneAgentSpec()
		assert.Truef(t, hasSpecChanged(ds, oa), ".image: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Image, oa.Image)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Image: "docker.io/dynatrace/oneagent",
		}}
		oa := newOneAgentSpec()
		oa.Image = "docker.io/dynatrace/oneagent"
		assert.Falsef(t, hasSpecChanged(ds, oa), ".image: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Image, oa.Image)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		oa.Image = "docker.io/dynatrace/oneagent"
		assert.Truef(t, hasSpecChanged(ds, oa), ".image: DaemonSet=%v OneAgent=%v", nil, oa.Image)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Image: "registry.access.redhat.com/dynatrace/oneagent",
		}}
		oa := newOneAgentSpec()
		oa.Image = "docker.io/dynatrace/oneagent"
		assert.Truef(t, hasSpecChanged(ds, oa), ".image: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Image, oa.Image)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Args: []string{"INFRA_ONLY=1"},
		}}
		oa := newOneAgentSpec()
		oa.Args = []string{"INFRA_ONLY=1"}
		assert.Falsef(t, hasSpecChanged(ds, oa), ".args: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Args, oa.Args)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Args: []string{"INFRA_ONLY=1"},
		}}
		oa := newOneAgentSpec()
		oa.Args = []string{"INFRA_ONLY=0"}
		assert.Truef(t, hasSpecChanged(ds, oa), ".args: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Args, oa.Args)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		oa.Args = []string{"INFRA_ONLY=0"}
		assert.Truef(t, hasSpecChanged(ds, oa), ".args: DaemonSet=%v OneAgent=%v", nil, oa.Args)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		oa.Resources = newResourceRequirements()
		assert.Truef(t, hasSpecChanged(ds, oa), ".resources: DaemonSet=%v OneAgent=%v", nil, oa.Resources)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Resources: newResourceRequirements(),
		}}
		assert.Truef(t, hasSpecChanged(ds, oa), ".resources: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Resources, nil)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		oa.PriorityClassName = "class"
		assert.Truef(t, hasSpecChanged(ds, oa), ".priorityClassName: DaemonSet=%v OneAgent=%v", nil, oa.PriorityClassName)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		ds.Template.Spec.PriorityClassName = "class"
		assert.Truef(t, hasSpecChanged(ds, oa), ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Template.Spec.PriorityClassName, nil)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.PriorityClassName = "class"
		oa := newOneAgentSpec()
		oa.PriorityClassName = "class"
		assert.Falsef(t, hasSpecChanged(ds, oa), ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Template.Spec.PriorityClassName, oa.PriorityClassName)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.PriorityClassName = "some class"
		oa := newOneAgentSpec()
		oa.PriorityClassName = "other class"
		assert.Truef(t, hasSpecChanged(ds, oa), ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Template.Spec.PriorityClassName, oa.PriorityClassName)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		oa.DNSPolicy = corev1.DNSClusterFirstWithHostNet
		assert.Truef(t, hasSpecChanged(ds, oa), ".dnsPolicy: DaemonSet=%v OneAgent=%v", ds.Template.Spec.DNSPolicy, oa.DNSPolicy)
	}
}

func TestCopyDaemonSetSpecToOneAgentSpec(t *testing.T) {
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		desired := newOneAgentSpec()

		copyDaemonSetSpecToOneAgentSpec(ds, oa)

		assert.Truef(t, reflect.DeepEqual(desired, oa), "empty daemonset")
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		desired := newOneAgentSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Image:     "docker.io/dynatrace/oneagent",
			Args:      []string{"INFRO_ONLY=1"},
			Resources: newResourceRequirements(),
			Env:       newEnvVar(),
		}}
		ds.Template.Spec.Tolerations = []corev1.Toleration{}
		ds.Template.Spec.NodeSelector = map[string]string{"k": "v"}
		ds.Template.Spec.PriorityClassName = "class"

		copyDaemonSetSpecToOneAgentSpec(ds, oa)

		assert.Falsef(t, reflect.DeepEqual(desired, oa), "non-empty daemonset")
		assert.Equalf(t, oa.Image, ds.Template.Spec.Containers[0].Image, ".image: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Image, oa.Image)
		assert.Equalf(t, oa.Args, ds.Template.Spec.Containers[0].Args, ".args: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Args, oa.Args)
		assert.Equalf(t, oa.Tolerations, ds.Template.Spec.Tolerations, ".tolerations: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Tolerations, oa.Tolerations)
		assert.Equalf(t, oa.NodeSelector, ds.Template.Spec.NodeSelector, ".nodeSelector: DaemonSet=%v OneAgent=%v", ds.Template.Spec.NodeSelector, oa.NodeSelector)
		assert.Equalf(t, oa.PriorityClassName, ds.Template.Spec.PriorityClassName, ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Template.Spec.PriorityClassName, oa.PriorityClassName)
		assert.Truef(t, reflect.DeepEqual(oa.Resources, ds.Template.Spec.Containers[0].Resources), ".resources: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Resources, oa.Resources)
	}
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
	oa.Status.Items = map[string]api.OneAgentInstance{"node-3": {Version: "outdated"}}
	doomed, instances := getPodsToRestart(pods, dtc, oa)
	assert.Lenf(t, doomed, 1, "list of pods to restart")
	assert.Equalf(t, doomed[0], pods[1], "list of pods to restart")
	assert.Lenf(t, instances, 3, "list of instances")
	assert.Equalf(t, instances["node-3"].Version, oa.Status.Items["node-3"].Version, "determine agent version from dynatrace server")
}

func TestNotifyDynatraceAboutMarkForTerminationEvent(t *testing.T) {

}

func newOneAgent() *api.OneAgent {
	return &api.OneAgent{
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

func newOneAgentSpec() *api.OneAgentSpec {
	return &api.OneAgentSpec{}
}

func newDaemonSetSpec() *appsv1.DaemonSetSpec {
	return &appsv1.DaemonSetSpec{}
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

func newEnvVar() []corev1.EnvVar {
	return []corev1.EnvVar{{
		Name:  "ONEAGENT_ENABLE_VOLUME_STORAGE",
		Value: "true",
	}}
}

func parseQuantity(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}
