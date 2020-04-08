package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	ns, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
	}

	m.GetWebhookServer().Register("/inject", &webhook.Admission{Handler: &podInjector{
		namespace: ns,
	}})
	return nil
}

var logger = log.Log.WithName("oneagent.webhook")

// podAnnotator injects the OneAgent into Pods
type podInjector struct {
	client    client.Client
	apiReader client.Reader
	decoder   *admission.Decoder
	namespace string
}

// podAnnotator adds an annotation to every incoming pods
func (m *podInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := m.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	pod.Annotations["oneagent.dynatrace.com/injected"] = "true"

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

// InjectAPIReader injects the API reader
func (m *podInjector) InjectAPIReader(c client.Reader) error {
	m.apiReader = c
	return nil
}

// InjectDecoder injects the decoder
func (m *podInjector) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}
