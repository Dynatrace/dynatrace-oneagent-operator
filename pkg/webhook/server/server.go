package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	dtwebhook "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/webhook"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var logger = log.Log.WithName("oneagent.webhook")

// AddToManager adds the Webhook server to the Manager
func AddToManager(mgr manager.Manager) error {
	ns, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
	}

	podName := os.Getenv(k8sutil.PodNameEnvVar)
	if podName == "" {
		logger.Info("No Pod name set for webhook container")
	}

	var pod corev1.Pod
	if err := mgr.GetAPIReader().Get(context.TODO(), client.ObjectKey{
		Name:      podName,
		Namespace: ns,
	}, &pod); err != nil {
		return err
	}

	mgr.GetWebhookServer().Register("/inject", &webhook.Admission{Handler: &podInjector{
		namespace: ns,
		image:     pod.Spec.Containers[0].Image,
	}})

	mgr.GetWebhookServer().Register("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	return nil
}

// podAnnotator injects the OneAgent into Pods
type podInjector struct {
	client    client.Client
	decoder   *admission.Decoder
	image     string
	namespace string
}

// podAnnotator adds an annotation to every incoming pods
func (m *podInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := m.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	logger.Info("injecting into Pod", "name", pod.Name, "generatedName", pod.GenerateName, "namespace", req.Namespace)

	var ns corev1.Namespace
	if err := m.client.Get(ctx, client.ObjectKey{Name: req.Namespace}, &ns); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	oaName := getField(ns.Labels, dtwebhook.LabelInstance, "")
	if oaName == "" {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("no OneAgent instance set for namespace: %s", req.Namespace))
	}

	var oa dynatracev1alpha1.OneAgent
	if err := m.client.Get(ctx, client.ObjectKey{Name: oaName, Namespace: m.namespace}, &oa); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	inject := getField(ns.Annotations, dtwebhook.AnnotationInject, "true")
	inject = getField(pod.Annotations, dtwebhook.AnnotationInject, inject)
	if inject == "false" {
		return admission.Patched("")
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	pod.Annotations[dtwebhook.AnnotationInjected] = "true"
	flavor := url.QueryEscape(getField(pod.Annotations, dtwebhook.AnnotationFlavor, "default"))
	technologies := url.QueryEscape(getField(pod.Annotations, dtwebhook.AnnotationTechnologies, "all"))

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: "oneagent",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: "oneagent-config",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dtwebhook.SecretConfigName,
				},
			},
		},
		corev1.Volume{
			Name: "oneagent-podinfo",
			VolumeSource: corev1.VolumeSource{
				DownwardAPI: &corev1.DownwardAPIVolumeSource{
					Items: []corev1.DownwardAPIVolumeFile{
						{Path: "name", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}},
						{Path: "namespace", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
						{Path: "uid", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.uid"}},
						{Path: "labels", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.labels"}},
						{Path: "annotations", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"}},
					},
				},
			},
		})

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
		Name:    "install-oneagent",
		Image:   m.image,
		Command: []string{"/usr/bin/env"},
		Args:    []string{"bash", "/mnt/config/init.sh"},
		Env: []corev1.EnvVar{
			{Name: "FLAVOR", Value: flavor},
			{Name: "TECHNOLOGIES", Value: technologies},
			{
				Name: "NODENAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
				},
			},
			{
				Name: "NODEIP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.hostIP"},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "oneagent", MountPath: dtwebhook.PathOneAgentDir},
			{Name: "oneagent-config", MountPath: "/mnt/config"},
		},
	})

	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]

		c.VolumeMounts = append(c.VolumeMounts,
			corev1.VolumeMount{
				Name:      "oneagent",
				MountPath: "/etc/ld.so.preload",
				SubPath:   "ld.so.preload",
			},
			corev1.VolumeMount{Name: "oneagent", MountPath: dtwebhook.PathOneAgentDir},
			corev1.VolumeMount{Name: "oneagent-podinfo", MountPath: "/opt/dynatrace/oneagent/agent/conf/pod"})

		c.Env = append(c.Env,
			corev1.EnvVar{Name: "LD_PRELOAD", Value: dtwebhook.PathOneAgentProcessAgent},
			corev1.EnvVar{Name: "DT_CONTAINER_NAME", Value: c.Name},
			corev1.EnvVar{Name: "DT_CONTAINER_IMAGE", Value: c.Image})

		if oa.Spec.Proxy != nil && (oa.Spec.Proxy.Value != "" || oa.Spec.Proxy.ValueFrom != "") {
			c.Env = append(c.Env,
				corev1.EnvVar{
					Name: "DT_PROXY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: dtwebhook.SecretConfigName,
							},
							Key: "proxy",
						},
					},
				})
		}
	}

	marshaledPod, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// InjectClient injects the client
func (m *podInjector) InjectClient(c client.Client) error {
	m.client = c
	return nil
}

// InjectDecoder injects the decoder
func (m *podInjector) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}

func getField(values map[string]string, key, defaultValue string) string {
	if values == nil {
		return defaultValue
	}
	if x := values[key]; x != "" {
		return x
	}
	return defaultValue
}
