package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/controllers/utils"
	dtwebhook "github.com/Dynatrace/dynatrace-oneagent-operator/webhook"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var logger = log.Log.WithName("oneagent.webhook")

// AddToManager adds the Webhook server to the Manager
func AddToManager(mgr manager.Manager, ns string) error {
	podName := os.Getenv("POD_NAME")
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

	inject := utils.GetField(ns.Annotations, dtwebhook.AnnotationInject, "true")
	inject = utils.GetField(pod.Annotations, dtwebhook.AnnotationInject, inject)
	if inject == "false" {
		return admission.Patched("")
	}

	oaName := utils.GetField(ns.Labels, dtwebhook.LabelInstance, "")
	if oaName == "" {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("no OneAgentAPM instance set for namespace: %s", req.Namespace))
	}

	var oa dynatracev1alpha1.OneAgentAPM
	if err := m.client.Get(ctx, client.ObjectKey{Name: oaName, Namespace: m.namespace}, &oa); k8serrors.IsNotFound(err) {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf(
			"namespace '%s' is assigned to OneAgentAPM instance '%s' but doesn't exist", req.Namespace, oaName))
	} else if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	if pod.Annotations[dtwebhook.AnnotationInjected] == "true" {
		return admission.Patched("")
	}
	pod.Annotations[dtwebhook.AnnotationInjected] = "true"

	flavor := getFlavor(oa.Spec.Flavor, pod.Annotations)
	technologies := url.QueryEscape(utils.GetField(pod.Annotations, dtwebhook.AnnotationTechnologies, "all"))
	installPath := utils.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)
	installerURL := utils.GetField(pod.Annotations, dtwebhook.AnnotationInstallerUrl, "")
	imageAnnotation := utils.GetField(pod.Annotations, dtwebhook.AnnotationImage, "")
	failurePolicy := utils.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, "silent")
	image := m.image

	if installerURL == "" && oa.Status.UseImmutableImage {
		if oa.Spec.Image == "" && imageAnnotation == "" {
			image, err = utils.BuildOneAgentAPMImage(oa.Spec.APIURL, flavor, technologies, oa.Spec.AgentVersion)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		} else if imageAnnotation != "" {
			image = imageAnnotation
		} else if oa.Spec.Image != "" {
			image = oa.Spec.Image
		}
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: "init",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
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
		})

	var sc *corev1.SecurityContext
	if pod.Spec.Containers[0].SecurityContext != nil {
		sc = pod.Spec.Containers[0].SecurityContext.DeepCopy()
	}

	if oa.Spec.Image == "" && imageAnnotation == "" && oa.Status.UseImmutableImage {
		pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: dtwebhook.PullSecretName})
	}

	fieldEnvVar := func(key string) *corev1.EnvVarSource {
		return &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: key}}
	}

	basePodName := pod.GenerateName
	if basePodName == "" {
		basePodName = pod.Name
	}

	// Only include up to the last dash character, exclusive.
	if p := strings.LastIndex(basePodName, "-"); p != -1 {
		basePodName = basePodName[:p]
	}

	useImmutableImage := ""
	if oa.Status.UseImmutableImage {
		useImmutableImage = "true"
	}

	ic := corev1.Container{
		Name:            "install-oneagent",
		Image:           image,
		ImagePullPolicy: corev1.PullAlways,
		Command:         []string{"/usr/bin/env"},
		Args:            []string{"bash", "/mnt/config/init.sh"},
		Env: []corev1.EnvVar{
			{Name: "FLAVOR", Value: flavor},
			{Name: "TECHNOLOGIES", Value: technologies},
			{Name: "INSTALLPATH", Value: installPath},
			{Name: "INSTALLER_URL", Value: installerURL},
			{Name: "FAILURE_POLICY", Value: failurePolicy},
			{Name: "CONTAINERS_COUNT", Value: strconv.Itoa(len(pod.Spec.Containers))},
			{Name: "K8S_PODNAME", ValueFrom: fieldEnvVar("metadata.name")},
			{Name: "K8S_PODUID", ValueFrom: fieldEnvVar("metadata.uid")},
			{Name: "K8S_BASEPODNAME", Value: basePodName},
			{Name: "K8S_NAMESPACE", ValueFrom: fieldEnvVar("metadata.namespace")},
			{Name: "K8S_NODE_NAME", ValueFrom: fieldEnvVar("spec.nodeName")},
			{Name: "USE_IMMUTABLE_IMAGE", Value: useImmutableImage},
		},
		SecurityContext: sc,
		VolumeMounts: []corev1.VolumeMount{
			{Name: "init", MountPath: "/mnt/init"},
			{Name: "oneagent", MountPath: "/mnt/oneagent"},
			{Name: "oneagent-config", MountPath: "/mnt/config"},
		},
		Resources: oa.Spec.Resources,
	}

	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]

		ic.Env = append(ic.Env,
			corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_NAME", i+1), Value: c.Name},
			corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_IMAGE", i+1), Value: c.Image})

		c.VolumeMounts = append(c.VolumeMounts,
			corev1.VolumeMount{
				Name:      "oneagent",
				MountPath: "/etc/ld.so.preload",
				SubPath:   "ld.so.preload",
			},
			corev1.VolumeMount{Name: "oneagent", MountPath: installPath},
			corev1.VolumeMount{
				Name:      "oneagent",
				MountPath: "/var/lib/dynatrace/oneagent/agent/config/container.conf",
				SubPath:   fmt.Sprintf("container_%s.conf", c.Name),
			})

		c.Env = append(c.Env,
			corev1.EnvVar{Name: "LD_PRELOAD", Value: installPath + "/agent/lib64/liboneagentproc.so"})

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

		if oa.Spec.NetworkZone != "" {
			c.Env = append(c.Env, corev1.EnvVar{Name: "DT_NETWORK_ZONE", Value: oa.Spec.NetworkZone})
		}
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, ic)

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
