package v1alpha1

import (
	"errors"
	"reflect"
	"testing"

	api "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MyDynatraceClient struct {
	mock.Mock
}

func (o *MyDynatraceClient) GetVersionForIp(ip string) (string, error) {
	args := o.Called(ip)
	return args.String(0), args.Error(1)
}

func (o *MyDynatraceClient) GetVersionForLatest(os, installerType string) (string, error) {
	args := o.Called(os, installerType)
	return args.String(0), args.Error(1)
}

func TestOneAgent_Validate(t *testing.T) {
	oa := newOneAgent()
	assert.Error(t, Validate(oa))
	oa.Spec.ApiUrl = "https://f.q.d.n/api"
	assert.NoError(t, Validate(oa))
}

func TestHasSpecChanged(t *testing.T) {
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		assert.Falsef(t, HasSpecChanged(ds, oa), "empty specs change detected")
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Image: "docker.io/dynatrace/oneagent",
		}}
		oa := newOneAgentSpec()
		assert.Truef(t, HasSpecChanged(ds, oa), ".image: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Image, oa.Image)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Image: "docker.io/dynatrace/oneagent",
		}}
		oa := newOneAgentSpec()
		oa.Image = "docker.io/dynatrace/oneagent"
		assert.Falsef(t, HasSpecChanged(ds, oa), ".image: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Image, oa.Image)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		oa.Image = "docker.io/dynatrace/oneagent"
		assert.Truef(t, HasSpecChanged(ds, oa), ".image: DaemonSet=%v OneAgent=%v", nil, oa.Image)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Image: "registry.access.redhat.com/dynatrace/oneagent",
		}}
		oa := newOneAgentSpec()
		oa.Image = "docker.io/dynatrace/oneagent"
		assert.Truef(t, HasSpecChanged(ds, oa), ".image: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Image, oa.Image)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Args: []string{"INFRA_ONLY=1"},
		}}
		oa := newOneAgentSpec()
		oa.Args = []string{"INFRA_ONLY=1"}
		assert.Falsef(t, HasSpecChanged(ds, oa), ".args: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Args, oa.Args)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Args: []string{"INFRA_ONLY=1"},
		}}
		oa := newOneAgentSpec()
		oa.Args = []string{"INFRA_ONLY=0"}
		assert.Truef(t, HasSpecChanged(ds, oa), ".args: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Args, oa.Args)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		oa.Args = []string{"INFRA_ONLY=0"}
		assert.Truef(t, HasSpecChanged(ds, oa), ".args: DaemonSet=%v OneAgent=%v", nil, oa.Args)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		oa.Resources = newResourceRequirements()
		assert.Truef(t, HasSpecChanged(ds, oa), ".resources: DaemonSet=%v OneAgent=%v", nil, oa.Resources)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		ds.Template.Spec.Containers = []corev1.Container{{
			Resources: newResourceRequirements(),
		}}
		assert.Truef(t, HasSpecChanged(ds, oa), ".resources: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Resources, nil)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		oa.PriorityClassName = "class"
		assert.Truef(t, HasSpecChanged(ds, oa), ".priorityClassName: DaemonSet=%v OneAgent=%v", nil, oa.PriorityClassName)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		ds.Template.Spec.PriorityClassName = "class"
		assert.Truef(t, HasSpecChanged(ds, oa), ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Template.Spec.PriorityClassName, nil)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.PriorityClassName = "class"
		oa := newOneAgentSpec()
		oa.PriorityClassName = "class"
		assert.Falsef(t, HasSpecChanged(ds, oa), ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Template.Spec.PriorityClassName, oa.PriorityClassName)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.PriorityClassName = "some class"
		oa := newOneAgentSpec()
		oa.PriorityClassName = "other class"
		assert.Truef(t, HasSpecChanged(ds, oa), ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Template.Spec.PriorityClassName, oa.PriorityClassName)
	}
}

func TestCopyDaemonSetSpecToOneAgentSpec(t *testing.T) {
	{
		ds := newDaemonSetSpec()
		oa := newOneAgentSpec()
		desired := newOneAgentSpec()

		CopyDaemonSetSpecToOneAgentSpec(ds, oa)

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
		}}
		ds.Template.Spec.Tolerations = []corev1.Toleration{}
		ds.Template.Spec.NodeSelector = map[string]string{"k": "v"}
		ds.Template.Spec.PriorityClassName = "class"

		CopyDaemonSetSpecToOneAgentSpec(ds, oa)

		assert.Falsef(t, reflect.DeepEqual(desired, oa), "non-empty daemonset")
		assert.Equalf(t, oa.Image, ds.Template.Spec.Containers[0].Image, ".image: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Image, oa.Image)
		assert.Equalf(t, oa.Args, ds.Template.Spec.Containers[0].Args, ".args: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Args, oa.Args)
		assert.Equalf(t, oa.Tolerations, ds.Template.Spec.Tolerations, ".tolerations: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Tolerations, oa.Tolerations)
		assert.Equalf(t, oa.NodeSelector, ds.Template.Spec.NodeSelector, ".nodeSelector: DaemonSet=%v OneAgent=%v", ds.Template.Spec.NodeSelector, oa.NodeSelector)
		assert.Equalf(t, oa.PriorityClassName, ds.Template.Spec.PriorityClassName, ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Template.Spec.PriorityClassName, oa.PriorityClassName)
		assert.Truef(t, reflect.DeepEqual(oa.Resources, ds.Template.Spec.Containers[0].Resources), ".resources: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Resources, oa.Resources)
	}
}

func TestApplyOneAgentSettings(t *testing.T) {
	{
		ds := newDaemonSet()
		oa := newOneAgent()
		ApplyOneAgentSettings(ds, oa)
		assert.Equalf(t, ds.Spec.Template.Spec.Containers[0].Image, oa.Spec.Image, ".image: DaemonSet=%v OneAgent=%v", ds.Spec.Template.Spec.Containers[0].Image, oa.Spec.Image)
		assert.Equalf(t, ds.Spec.Template.Spec.Containers[0].Args, oa.Spec.Args, ".args: DaemonSet=%v OneAgent=%v", ds.Spec.Template.Spec.Containers[0].Args, oa.Spec.Args)
		assert.Equalf(t, ds.Spec.Template.Spec.Tolerations, oa.Spec.Tolerations, ".tolerations: DaemonSet=%v OneAgent=%v", ds.Spec.Template.Spec.Tolerations, oa.Spec.Tolerations)
		assert.Equalf(t, ds.Spec.Template.Spec.NodeSelector, oa.Spec.NodeSelector, ".nodeSelector: DaemonSet=%v OneAgent=%v", ds.Spec.Template.Spec.NodeSelector, oa.Spec.NodeSelector)
		labels := util.BuildLabels(oa.Name)
		assert.Truef(t, reflect.DeepEqual(ds.ObjectMeta.Labels, labels), ".ObjectMeta.Labels mismatch")
		assert.Truef(t, reflect.DeepEqual(ds.Spec.Selector.MatchLabels, labels), ".Spec.Selector.MatchLabels mismatch")
		assert.Truef(t, reflect.DeepEqual(ds.Spec.Template.ObjectMeta.Labels, labels), ".Spec.Template.ObjectMeta.Labels mismatch")
		assert.Truef(t, reflect.DeepEqual(ds.Spec.Template.Spec.Containers[0].Resources, oa.Spec.Resources), ".resources: DaemonSet=%v OneAgent=%v", ds.Spec.Template.Spec.Containers[0].Resources, oa.Spec.Resources)
	}
	{
		ds := newDaemonSet()
		oa := newOneAgent()
		oa.Spec = api.OneAgentSpec{
			Image:             "docker.io/dynatrace/oneagent",
			Args:              []string{"INFRO_ONLY=1"},
			Tolerations:       []corev1.Toleration{},
			NodeSelector:      map[string]string{"k": "v"},
			Resources:         newResourceRequirements(),
			PriorityClassName: "class",
		}
		ApplyOneAgentSettings(ds, oa)
		assert.Equalf(t, ds.Spec.Template.Spec.Containers[0].Image, oa.Spec.Image, ".image: DaemonSet=%v OneAgent=%v", ds.Spec.Template.Spec.Containers[0].Image, oa.Spec.Image)
		assert.Equalf(t, ds.Spec.Template.Spec.Containers[0].Args, oa.Spec.Args, ".args: DaemonSet=%v OneAgent=%v", ds.Spec.Template.Spec.Containers[0].Args, oa.Spec.Args)
		assert.Equalf(t, ds.Spec.Template.Spec.Tolerations, oa.Spec.Tolerations, ".tolerations: DaemonSet=%v OneAgent=%v", ds.Spec.Template.Spec.Tolerations, oa.Spec.Tolerations)
		assert.Equalf(t, ds.Spec.Template.Spec.NodeSelector, oa.Spec.NodeSelector, ".nodeSelector: DaemonSet=%v OneAgent=%v", ds.Spec.Template.Spec.NodeSelector, oa.Spec.NodeSelector)
		assert.Equalf(t, ds.Spec.Template.Spec.PriorityClassName, oa.Spec.PriorityClassName, ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Spec.Template.Spec.PriorityClassName, oa.Spec.PriorityClassName)
		labels := util.BuildLabels(oa.Name)
		assert.Truef(t, reflect.DeepEqual(ds.ObjectMeta.Labels, labels), ".ObjectMeta.Labels mismatch")
		assert.Truef(t, reflect.DeepEqual(ds.Spec.Selector.MatchLabels, labels), ".Spec.Selector.MatchLabels mismatch")
		assert.Truef(t, reflect.DeepEqual(ds.Spec.Template.ObjectMeta.Labels, labels), ".Spec.Template.ObjectMeta.Labels mismatch")
		assert.Truef(t, reflect.DeepEqual(ds.Spec.Template.Spec.Containers[0].Resources, oa.Spec.Resources), ".resources: DaemonSet=%v OneAgent=%v", ds.Spec.Template.Spec.Containers[0].Resources, oa.Spec.Resources)
	}
}

func TestApplyOneAgentDefaults(t *testing.T) {
	ds := newDaemonSet()
	oa := newOneAgent()
	ApplyOneAgentDefaults(ds, oa)
	assert.Equalf(t, ds.Spec.Template.Spec.HostNetwork, true, ".Spec.Template.Spec.HostNetwork")
	assert.Equalf(t, ds.Spec.Template.Spec.HostPID, true, ".Spec.Template.Spec.HostPID")
	assert.Equalf(t, ds.Spec.Template.Spec.HostIPC, true, ".Spec.Template.Spec.HostIPC")
	assert.Equalf(t, ds.Spec.Template.Spec.ServiceAccountName, "dynatrace-oneagent", ".Spec.Template.Spec.ServiceAccountName")
}

func TestGetPodsToRestart(t *testing.T) {
	dtc := new(MyDynatraceClient)
	dtc.On("GetVersionForIp", "127.0.0.1").Return("1.2.3", nil)
	dtc.On("GetVersionForIp", "127.0.0.2").Return("0.1.2", nil)
	dtc.On("GetVersionForIp", "127.0.0.3").Return("", errors.New("n/a"))

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
	doomed, instances := GetPodsToRestart(pods, dtc, oa)
	assert.Lenf(t, doomed, 1, "list of pods to restart")
	assert.Equalf(t, doomed[0], pods[1], "list of pods to restart")
	assert.Lenf(t, instances, 3, "list of instances")
	assert.Equalf(t, instances["node-3"].Version, oa.Status.Items["node-3"].Version, "determine agent version from dynatrace server")
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

func newDaemonSet() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-daemonset",
			Namespace: "my-namespace",
			UID:       "ed8b7bb5-9bbf-4019-baf2-7493062e03d3",
		},
	}
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

func parseQuantity(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}
