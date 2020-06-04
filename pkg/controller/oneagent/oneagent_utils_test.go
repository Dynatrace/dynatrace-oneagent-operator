package oneagent

import (
	"errors"
	"testing"

	api "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
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

func TestHasSpecChanged(t *testing.T) {
	{
		ds := newDaemonSetSpec()
		exp := &newDaemonSetForCR(newOneAgent()).Spec
		assert.Falsef(t, hasSpecChanged(ds, exp), "empty specs change detected")
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers[0].Image = "docker.io/dynatrace/oneagent"
		exp := &newDaemonSetForCR(newOneAgent()).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".image: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Image, exp.Template.Spec.Containers[0].Image)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers[0].Image = "docker.io/dynatrace/oneagent"
		oa := newOneAgent()
		oa.Spec.Image = "docker.io/dynatrace/oneagent"
		exp := &newDaemonSetForCR(oa).Spec
		assert.Falsef(t, hasSpecChanged(ds, exp), ".image: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Image, exp.Template.Spec.Containers[0].Image)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgent()
		oa.Spec.Image = "docker.io/dynatrace/oneagent"
		exp := &newDaemonSetForCR(oa).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".image: DaemonSet=%v OneAgent=%v", nil, exp.Template.Spec.Containers[0].Image)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers[0].Image = "registry.access.redhat.com/dynatrace/oneagent"
		oa := newOneAgent()
		oa.Spec.Image = "docker.io/dynatrace/oneagent"
		exp := &newDaemonSetForCR(oa).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".image: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Image, exp.Template.Spec.Containers[0].Image)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers[0].Args = []string{"INFRA_ONLY=1", "--set-host-property=OperatorVersion=snapshot"}
		oa := newOneAgent()
		oa.Spec.Args = []string{"INFRA_ONLY=1"}
		exp := &newDaemonSetForCR(oa).Spec
		assert.Falsef(t, hasSpecChanged(ds, exp), ".args: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Args, exp.Template.Spec.Containers[0].Args)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers[0].Args = []string{"INFRA_ONLY=1"}
		oa := newOneAgent()
		oa.Spec.Args = []string{"INFRA_ONLY=0"}
		exp := &newDaemonSetForCR(oa).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".args: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Args, exp.Template.Spec.Containers[0].Args)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgent()
		oa.Spec.Args = []string{"INFRA_ONLY=0"}
		exp := &newDaemonSetForCR(oa).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".args: DaemonSet=%v OneAgent=%v", nil, exp.Template.Spec.Containers[0].Args)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgent()
		oa.Spec.Resources = newResourceRequirements()
		exp := &newDaemonSetForCR(oa).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".resources: DaemonSet=%v OneAgent=%v", nil, exp.Template.Spec.Containers[0].Resources)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgent()
		ds.Template.Spec.Containers[0].Resources = newResourceRequirements()
		exp := &newDaemonSetForCR(oa).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".resources: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].Resources, nil)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Containers[0].VolumeMounts = nil
		oa := newOneAgent()
		exp := &newDaemonSetForCR(oa).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".resources: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Containers[0].VolumeMounts, exp.Template.Spec.Containers[0].VolumeMounts)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgent()
		oa.Spec.PriorityClassName = "class"
		exp := &newDaemonSetForCR(oa).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".priorityClassName: DaemonSet=%v OneAgent=%v", nil, exp.Template.Spec.PriorityClassName)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.PriorityClassName = "class"
		exp := &newDaemonSetForCR(newOneAgent()).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Template.Spec.PriorityClassName, nil)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.PriorityClassName = "class"
		oa := newOneAgent()
		oa.Spec.PriorityClassName = "class"
		exp := &newDaemonSetForCR(oa).Spec
		assert.Falsef(t, hasSpecChanged(ds, exp), ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Template.Spec.PriorityClassName, exp.Template.Spec.PriorityClassName)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.PriorityClassName = "some class"
		oa := newOneAgent()
		oa.Spec.PriorityClassName = "other class"
		exp := &newDaemonSetForCR(oa).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".priorityClassName: DaemonSet=%v OneAgent=%v", ds.Template.Spec.PriorityClassName, exp.Template.Spec.PriorityClassName)
	}
	{
		ds := newDaemonSetSpec()
		oa := newOneAgent()
		oa.Spec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
		exp := &newDaemonSetForCR(oa).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".dnsPolicy: DaemonSet=%v OneAgent=%v", ds.Template.Spec.DNSPolicy, exp.Template.Spec.DNSPolicy)
	}
	{
		ds := newDaemonSetSpec()
		ds.Template.Spec.Volumes = nil
		oa := newOneAgent()
		exp := &newDaemonSetForCR(oa).Spec
		assert.Truef(t, hasSpecChanged(ds, exp), ".volumes: DaemonSet=%v OneAgent=%v", ds.Template.Spec.Volumes, exp.Template.Spec.Volumes)
	}
}

func TestGetPodsToRestart(t *testing.T) {
	dtc := new(dtclient.MockDynatraceClient)
	dtc.On("GetAgentVersionForIP", "127.0.0.1", "").Return("1.2.3", nil)
	dtc.On("GetAgentVersionForIP", "127.0.0.2", "").Return("0.1.2", nil)
	dtc.On("GetAgentVersionForIP", "127.0.0.3", "").Return("", errors.New("n/a"))

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
	oa.Status.Instances = map[string]api.OneAgentInstance{"node-3": {Version: "outdated"}}
	doomed, instances, err := getPodsToRestart(pods, dtc, oa)
	assert.Lenf(t, doomed, 1, "list of pods to restart")
	assert.Equalf(t, doomed[0], pods[1], "list of pods to restart")
	assert.Lenf(t, instances, 3, "list of instances")
	assert.Equalf(t, instances["node-3"].Version, oa.Status.Instances["node-3"].Version, "determine agent version from dynatrace server")
	assert.Equal(t, nil, err)
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
	return &appsv1.DaemonSetSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"dynatrace": "oneagent",
					"oneagent":  "my-oneagent",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "dynatrace-oneagent",
				Containers: []corev1.Container{
					{
						Image: "docker.io/dynatrace/oneagent:latest",
						Args: []string{
							"--set-host-property=OperatorVersion=snapshot",
						},
						Env: []corev1.EnvVar{
							{
								Name: "ONEAGENT_INSTALLER_TOKEN",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "my-oneagent"},
										Key:                  utils.DynatracePaasToken,
									},
								},
							},
							{
								Name:  "ONEAGENT_INSTALLER_SCRIPT_URL",
								Value: "/v1/deployment/installer/agent/unix/default/latest?Api-Token=$(ONEAGENT_INSTALLER_TOKEN)&arch=x86&flavor=default",
							},
							{
								Name:  "ONEAGENT_INSTALLER_SKIP_CERT_CHECK",
								Value: "false",
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "host-root",
								MountPath: "/mnt/root",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "host-root",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/",
							},
						},
					},
				},
			},
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
