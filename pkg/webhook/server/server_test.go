package server

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	dtwebhook "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/webhook"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func init() {
	apis.AddToScheme(scheme.Scheme)
}

func TestPodInjection(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj := &podInjector{
		client: fake.NewFakeClient(
			&dynatracev1alpha1.OneAgentAPM{
				ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
				Spec: dynatracev1alpha1.OneAgentAPMSpec{BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
					APIURL: "https://test-api-url.com/api",
				}},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
				},
			},
		),
		decoder:   decoder,
		image:     "test-image",
		namespace: "dynatrace",
	}

	basePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod-123456", Namespace: "test-namespace"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
			}},
		},
	}
	basePodBytes, err := json.Marshal(&basePod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1beta1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: basePodBytes,
			},
			Namespace: "test-namespace",
		},
	}
	resp := inj.Handle(context.TODO(), req)
	require.NoError(t, resp.Complete(req))

	if !resp.Allowed {
		require.FailNow(t, "failed to inject", resp.Result)
	}

	patchType := admissionv1beta1.PatchTypeJSONPatch
	assert.Equal(t, resp.PatchType, &patchType)

	patch, err := jsonpatch.DecodePatch(resp.Patch)
	require.NoError(t, err)

	updPodBytes, err := patch.Apply(basePodBytes)
	require.NoError(t, err)

	var updPod corev1.Pod
	require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

	assert.Equal(t, corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-123456",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				"oneagent.dynatrace.com/injected": "true",
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name:    "install-oneagent",
				Image:   "test-api-url.com/linux/codemodule",
				Command: []string{"/usr/bin/env"},
				Args:    []string{"bash", "/mnt/config/init.sh"},
				Env: []corev1.EnvVar{
					{Name: "FLAVOR", Value: "default"},
					{Name: "TECHNOLOGIES", Value: "all"},
					{Name: "INSTALLPATH", Value: "/opt/dynatrace/oneagent-paas"},
					{Name: "INSTALLER_URL", Value: ""},
					{Name: "FAILURE_POLICY", Value: "silent"},
					{Name: "CONTAINERS_COUNT", Value: "1"},
					{Name: "K8S_PODNAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
					{Name: "K8S_PODUID", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.uid"}}},
					{Name: "K8S_BASEPODNAME", Value: "test-pod"},
					{Name: "K8S_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
					{Name: "K8S_NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
					{Name: "CONTAINER_1_NAME", Value: "test-container"},
					{Name: "CONTAINER_1_IMAGE", Value: "alpine"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent", MountPath: "/mnt/oneagent"},
					{Name: "oneagent-config", MountPath: "/mnt/config"},
				},
			}},
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
				Env: []corev1.EnvVar{
					{Name: "LD_PRELOAD", Value: "/opt/dynatrace/oneagent-paas/agent/lib64/liboneagentproc.so"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent", MountPath: "/etc/ld.so.preload", SubPath: "ld.so.preload"},
					{Name: "oneagent", MountPath: "/opt/dynatrace/oneagent-paas"},
					{
						Name:      "oneagent",
						MountPath: "/var/lib/dynatrace/oneagent/agent/config/container.conf",
						SubPath:   "container_test-container.conf",
					},
				},
			}},
			Volumes: []corev1.Volume{
				{
					Name: "oneagent",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "oneagent-config",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: dtwebhook.SecretConfigName,
						},
					},
				},
			},
			ImagePullSecrets: []corev1.LocalObjectReference{
				{
					Name: "dynatrace-oneagent-pull-secret",
				},
			},
		},
	}, updPod)
}

func TestPodInjectionWithImage(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj := &podInjector{
		client: fake.NewFakeClient(
			&dynatracev1alpha1.OneAgentAPM{
				ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
				Spec:       dynatracev1alpha1.OneAgentAPMSpec{Image: "customregistry/linux/codemodule"},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
				},
			},
		),
		decoder:   decoder,
		image:     "test-image",
		namespace: "dynatrace",
	}

	basePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "test-namespace"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
			}},
		},
	}
	basePodBytes, err := json.Marshal(&basePod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1beta1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: basePodBytes,
			},
			Namespace: "test-namespace",
		},
	}
	resp := inj.Handle(context.TODO(), req)
	require.NoError(t, resp.Complete(req))

	if !resp.Allowed {
		require.FailNow(t, "failed to inject", resp.Result)
	}

	patchType := admissionv1beta1.PatchTypeJSONPatch
	assert.Equal(t, resp.PatchType, &patchType)

	patch, err := jsonpatch.DecodePatch(resp.Patch)
	require.NoError(t, err)

	updPodBytes, err := patch.Apply(basePodBytes)
	require.NoError(t, err)

	var updPod corev1.Pod
	require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

	assert.Equal(t, corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				"oneagent.dynatrace.com/injected": "true",
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name:    "install-oneagent",
				Image:   "customregistry/linux/codemodule",
				Command: []string{"/usr/bin/env"},
				Args:    []string{"bash", "/mnt/config/init.sh"},
				Env: []corev1.EnvVar{
					{Name: "FLAVOR", Value: "default"},
					{Name: "TECHNOLOGIES", Value: "all"},
					{Name: "INSTALLPATH", Value: "/opt/dynatrace/oneagent-paas"},
					{Name: "INSTALLER_URL", Value: ""},
					{Name: "FAILURE_POLICY", Value: "silent"},
					{Name: "CONTAINERS_COUNT", Value: "1"},
					{Name: "K8S_PODNAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
					{Name: "K8S_PODUID", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.uid"}}},
					{Name: "K8S_BASEPODNAME", Value: "test"},
					{Name: "K8S_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
					{Name: "K8S_NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
					{Name: "CONTAINER_1_NAME", Value: "test-container"},
					{Name: "CONTAINER_1_IMAGE", Value: "alpine"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent", MountPath: "/mnt/oneagent"},
					{Name: "oneagent-config", MountPath: "/mnt/config"},
				},
			}},
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
				Env: []corev1.EnvVar{
					{Name: "LD_PRELOAD", Value: "/opt/dynatrace/oneagent-paas/agent/lib64/liboneagentproc.so"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent", MountPath: "/etc/ld.so.preload", SubPath: "ld.so.preload"},
					{Name: "oneagent", MountPath: "/opt/dynatrace/oneagent-paas"},
					{
						Name:      "oneagent",
						MountPath: "/var/lib/dynatrace/oneagent/agent/config/container.conf",
						SubPath:   "container_test-container.conf",
					},
				},
			}},
			Volumes: []corev1.Volume{
				{
					Name: "oneagent",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "oneagent-config",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: dtwebhook.SecretConfigName,
						},
					},
				},
			},
		},
	}, updPod)
}

func TestPodInjectionWithImageAnnotation(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj := &podInjector{
		client: fake.NewFakeClient(
			&dynatracev1alpha1.OneAgentAPM{
				ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
				Spec:       dynatracev1alpha1.OneAgentAPMSpec{},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
				},
			},
		),
		decoder:   decoder,
		image:     "test-image",
		namespace: "dynatrace",
	}

	basePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				"oneagent.dynatrace.com/image": "customregistry/linux/codemodule",
			}},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
			}},
		},
	}
	basePodBytes, err := json.Marshal(&basePod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1beta1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: basePodBytes,
			},
			Namespace: "test-namespace",
		},
	}
	resp := inj.Handle(context.TODO(), req)
	require.NoError(t, resp.Complete(req))

	if !resp.Allowed {
		require.FailNow(t, "failed to inject", resp.Result)
	}

	patchType := admissionv1beta1.PatchTypeJSONPatch
	assert.Equal(t, resp.PatchType, &patchType)

	patch, err := jsonpatch.DecodePatch(resp.Patch)
	require.NoError(t, err)

	updPodBytes, err := patch.Apply(basePodBytes)
	require.NoError(t, err)

	var updPod corev1.Pod
	require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

	assert.Equal(t, corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				"oneagent.dynatrace.com/injected": "true",
				"oneagent.dynatrace.com/image":    "customregistry/linux/codemodule",
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name:    "install-oneagent",
				Image:   "customregistry/linux/codemodule",
				Command: []string{"/usr/bin/env"},
				Args:    []string{"bash", "/mnt/config/init.sh"},
				Env: []corev1.EnvVar{
					{Name: "FLAVOR", Value: "default"},
					{Name: "TECHNOLOGIES", Value: "all"},
					{Name: "INSTALLPATH", Value: "/opt/dynatrace/oneagent-paas"},
					{Name: "INSTALLER_URL", Value: ""},
					{Name: "FAILURE_POLICY", Value: "silent"},
					{Name: "CONTAINERS_COUNT", Value: "1"},
					{Name: "K8S_PODNAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
					{Name: "K8S_PODUID", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.uid"}}},
					{Name: "K8S_BASEPODNAME", Value: "test"},
					{Name: "K8S_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
					{Name: "K8S_NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
					{Name: "CONTAINER_1_NAME", Value: "test-container"},
					{Name: "CONTAINER_1_IMAGE", Value: "alpine"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent", MountPath: "/mnt/oneagent"},
					{Name: "oneagent-config", MountPath: "/mnt/config"},
				},
			}},
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
				Env: []corev1.EnvVar{
					{Name: "LD_PRELOAD", Value: "/opt/dynatrace/oneagent-paas/agent/lib64/liboneagentproc.so"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent", MountPath: "/etc/ld.so.preload", SubPath: "ld.so.preload"},
					{Name: "oneagent", MountPath: "/opt/dynatrace/oneagent-paas"},
					{
						Name:      "oneagent",
						MountPath: "/var/lib/dynatrace/oneagent/agent/config/container.conf",
						SubPath:   "container_test-container.conf",
					},
				},
			}},
			Volumes: []corev1.Volume{
				{
					Name: "oneagent",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "oneagent-config",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: dtwebhook.SecretConfigName,
						},
					},
				},
			},
		},
	}, updPod)
}

func TestPodInjectionWithImageAnnotationOverwrite(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj := &podInjector{
		client: fake.NewFakeClient(
			&dynatracev1alpha1.OneAgentAPM{
				ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
				Spec:       dynatracev1alpha1.OneAgentAPMSpec{},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
				},
			},
		),
		decoder:   decoder,
		image:     "test-image",
		namespace: "dynatrace",
	}

	basePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				"oneagent.dynatrace.com/image":         "tenant123/linux/codemodule",
				"oneagent.dynatrace.com/installer-url": "installerurl",
			}},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
			}},
		},
	}
	basePodBytes, err := json.Marshal(&basePod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1beta1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: basePodBytes,
			},
			Namespace: "test-namespace",
		},
	}
	resp := inj.Handle(context.TODO(), req)
	require.NoError(t, resp.Complete(req))

	if !resp.Allowed {
		require.FailNow(t, "failed to inject", resp.Result)
	}

	patchType := admissionv1beta1.PatchTypeJSONPatch
	assert.Equal(t, resp.PatchType, &patchType)

	patch, err := jsonpatch.DecodePatch(resp.Patch)
	require.NoError(t, err)

	updPodBytes, err := patch.Apply(basePodBytes)
	require.NoError(t, err)

	var updPod corev1.Pod
	require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

	assert.Equal(t, corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				"oneagent.dynatrace.com/injected":      "true",
				"oneagent.dynatrace.com/image":         "tenant123/linux/codemodule",
				"oneagent.dynatrace.com/installer-url": "installerurl",
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name:    "install-oneagent",
				Image:   "test-image",
				Command: []string{"/usr/bin/env"},
				Args:    []string{"bash", "/mnt/config/init.sh"},
				Env: []corev1.EnvVar{
					{Name: "FLAVOR", Value: "default"},
					{Name: "TECHNOLOGIES", Value: "all"},
					{Name: "INSTALLPATH", Value: "/opt/dynatrace/oneagent-paas"},
					{Name: "INSTALLER_URL", Value: "installerurl"},
					{Name: "FAILURE_POLICY", Value: "silent"},
					{Name: "CONTAINERS_COUNT", Value: "1"},
					{Name: "K8S_PODNAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
					{Name: "K8S_PODUID", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.uid"}}},
					{Name: "K8S_BASEPODNAME", Value: "test"},
					{Name: "K8S_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
					{Name: "K8S_NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
					{Name: "CONTAINER_1_NAME", Value: "test-container"},
					{Name: "CONTAINER_1_IMAGE", Value: "alpine"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent", MountPath: "/mnt/oneagent"},
					{Name: "oneagent-config", MountPath: "/mnt/config"},
				},
			}},
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
				Env: []corev1.EnvVar{
					{Name: "LD_PRELOAD", Value: "/opt/dynatrace/oneagent-paas/agent/lib64/liboneagentproc.so"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent", MountPath: "/etc/ld.so.preload", SubPath: "ld.so.preload"},
					{Name: "oneagent", MountPath: "/opt/dynatrace/oneagent-paas"},
					{
						Name:      "oneagent",
						MountPath: "/var/lib/dynatrace/oneagent/agent/config/container.conf",
						SubPath:   "container_test-container.conf",
					},
				},
			}},
			Volumes: []corev1.Volume{
				{
					Name: "oneagent",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "oneagent-config",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: dtwebhook.SecretConfigName,
						},
					},
				},
			},
		},
	}, updPod)
}
