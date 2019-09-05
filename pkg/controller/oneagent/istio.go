package oneagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	versionedistioclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/clientset/versioned"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileOneAgent) reconcileIstio(logger logr.Logger, instance *dynatracev1alpha1.OneAgent, dtc dtclient.Client) (updated bool, ok bool) {
	var err error

	// Determine if cluster runs istio in default cluster
	enabled, err := istio.CheckIstioEnabled(r.config)
	if err != nil {
		logger.Error(err, "istio: failed to verify Istio availability")
		return false, false
	}

	logger.Info("istio: status", "enabled", enabled)

	if !enabled {
		return false, true
	}

	apiHost, err := dtc.GetAPIURLHost()
	if err != nil {
		logger.Error(err, "istio: failed to get host for Dynatrace API URL")
		return false, false
	}
	ic, err := r.initialiseIstioClient(logger)
	if err != nil {
		logger.Error(err, "istio: error initialising client for isitio")
		return false, false
	}

	upd, err := r.reconcileIstioConfigurations(instance, ic, []dtclient.CommunicationHost{apiHost}, "api-url", logger)
	if err != nil {
		logger.Error(err, "istio: error reconciling config for Dynatrace API URL")
		return false, false
	} else if upd {
		return true, true
	}

	// Fetch endpoints via Dynatrace client
	comHosts, err := dtc.GetCommunicationHosts()
	if err != nil {
		logger.Error(err, "istio: failed to get Dynatrace communication endpoints")
		return false, false
	}

	if upd, err := r.reconcileIstioConfigurations(instance, ic, comHosts, "communication-endpoint", logger); err != nil {
		logger.Error(err, "istio: error reconciling config for Dynatrace communication endpoints")
		return false, false
	} else if upd {
		return true, true
	}

	return false, true
}

func (r *ReconcileOneAgent) reconcileIstioConfigurations(
	instance *dynatracev1alpha1.OneAgent,
	ic *versionedistioclient.Clientset,
	comHosts []dtclient.CommunicationHost,
	role string,
	logger logr.Logger) (bool, error) {

	add := r.reconcileIstioCreateConfigurations(instance, comHosts, role, logger)
	rem := r.reconcileIstioRemoveConfigurations(instance, ic, comHosts, role, logger)
	return add || rem, nil
}

func (r *ReconcileOneAgent) reconcileIstioRemoveConfigurations(
	instance *dynatracev1alpha1.OneAgent,
	ic *versionedistioclient.Clientset,
	comHosts []dtclient.CommunicationHost,
	role string,
	logger logr.Logger) bool {

	labels := labels.SelectorFromSet(buildIstioLabels(instance.Name, role)).String()
	listOps := &metav1.ListOptions{
		LabelSelector: labels,
	}

	seen := map[string]bool{}
	for _, ch := range comHosts {
		seen[istio.BuildNameForEndpoint(instance.Name, ch.Host, ch.Port)] = true
	}

	vsUpd := r.removeIstioConfigurationForVirtualService(ic, listOps, seen, logger)
	seUpd := r.removeIstioConfigurationForServiceEntry(ic, listOps, seen, logger)

	return vsUpd || seUpd
}

func (r *ReconcileOneAgent) initialiseIstioClient(logger logr.Logger) (*versionedistioclient.Clientset, error) {
	ic, err := versionedistioclient.NewForConfig(r.config)
	if err != nil {
		logger.Error(err, fmt.Sprint("istio: failed to initialise client"))
	}
	return ic, err
}

func (r *ReconcileOneAgent) removeIstioConfigurationForServiceEntry(
	ic *versionedistioclient.Clientset,
	listOps *metav1.ListOptions,
	seen map[string]bool,
	logger logr.Logger) bool {

	gvk := istio.ServiceEntryGVK
	namespace := os.Getenv(k8sutil.WatchNamespaceEnvVar)

	list, err := ic.NetworkingV1alpha3().ServiceEntries(namespace).List(*listOps)
	if err != nil {
		logger.Error(err, fmt.Sprintf("istio: error listing service entries, %v", err))
		return false
	}

	del := false
	for _, se := range list.Items {
		if _, inUse := seen[se.GetName()]; !inUse {
			logger.Info(fmt.Sprintf("istio: removing %s: %v", gvk.Kind, se.GetName()))
			err = ic.NetworkingV1alpha3().
				ServiceEntries(namespace).
				Delete(se.GetName(), &metav1.DeleteOptions{})
			if err != nil {
				logger.Error(err, fmt.Sprintf("istio: error deleteing service entry, %s : %v", se.GetName(), err))
				continue
			}
			del = true
		}
	}
	return del

}

func (r *ReconcileOneAgent) removeIstioConfigurationForVirtualService(
	ic *versionedistioclient.Clientset,
	listOps *metav1.ListOptions,
	seen map[string]bool,
	logger logr.Logger) bool {

	gvk := istio.VirtualServiceGVK
	namespace := os.Getenv(k8sutil.WatchNamespaceEnvVar)

	list, err := ic.NetworkingV1alpha3().VirtualServices(namespace).List(*listOps)
	if err != nil {
		logger.Error(err, fmt.Sprintf("istio: error listing virtual service, %v", err))
		return false
	}

	del := false
	for _, vs := range list.Items {
		if _, inUse := seen[vs.GetName()]; !inUse {
			logger.Info(fmt.Sprintf("istio: removing %s: %v", gvk.Kind, vs.GetName()))
			err = ic.NetworkingV1alpha3().
				VirtualServices(namespace).
				Delete(vs.GetName(), &metav1.DeleteOptions{})
			if err != nil {
				logger.Error(err, fmt.Sprintf("istio: error deleteing virtual service, %s : %v", vs.GetName(), err))
				continue
			}
			del = true
		}
	}
	return del
}

func (r *ReconcileOneAgent) reconcileIstioCreateConfigurations(instance *dynatracev1alpha1.OneAgent,
	comHosts []dtclient.CommunicationHost, role string, logger logr.Logger) bool {

	created := false

	for _, ch := range comHosts {
		name := istio.BuildNameForEndpoint(instance.Name, ch.Host, ch.Port)

		if notFound := r.configurationExists(istio.ServiceEntryGVK, instance.Namespace, name); notFound {
			logger.Info("istio: creating ServiceEntry", "objectName", name, "host", ch.Host, "port", ch.Port)
			payload := istio.BuildServiceEntry(name, ch.Host, ch.Port, ch.Protocol)
			if err := r.reconcileIstioCreateConfiguration(instance, istio.ServiceEntryGVK, role, payload); err != nil {
				logger.Error(err, "istio: failed to create ServiceEntry")
				continue
			}
			created = true
		}

		if notFound := r.configurationExists(istio.VirtualServiceGVK, instance.Namespace, name); notFound {
			logger.Info("istio: creating VirtualService", "objectName", name, "host", ch.Host, "port", ch.Port, "protocol", ch.Protocol)
			payload := istio.BuildVirtualService(name, ch.Host, ch.Port, ch.Protocol)
			if err := r.reconcileIstioCreateConfiguration(instance, istio.VirtualServiceGVK, role, payload); err != nil {
				logger.Error(err, "istio: failed to create VirtualService")
			}
			created = true
		}
	}

	return created
}

func (r *ReconcileOneAgent) reconcileIstioCreateConfiguration(instance *dynatracev1alpha1.OneAgent,
	gvk schema.GroupVersionKind, role string, payload []byte) error {

	var obj unstructured.Unstructured
	obj.Object = make(map[string]interface{})

	if err := json.Unmarshal(payload, &obj.Object); err != nil {
		return fmt.Errorf("failed to unmarshal json (%s): %v", payload, err)
	}

	obj.SetGroupVersionKind(gvk)
	obj.SetLabels(buildIstioLabels(instance.Name, role))

	if err := controllerutil.SetControllerReference(instance, &obj, r.scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %v", err)
	}

	if err := r.client.Create(context.TODO(), &obj); err != nil {
		return fmt.Errorf("failed to create Istio configuration: %v", err)
	}

	return nil
}

func (r *ReconcileOneAgent) configurationExists(gvk schema.GroupVersionKind, namespace string, name string) bool {
	var objQuery unstructured.Unstructured
	objQuery.Object = make(map[string]interface{})

	objQuery.SetGroupVersionKind(gvk)
	key := client.ObjectKey{Namespace: namespace, Name: name}

	return errors.IsNotFound(r.client.Get(context.TODO(), key, &objQuery))
}

func buildIstioLabels(name, role string) map[string]string {
	labels := buildLabels(name)
	labels["dynatrace-istio-role"] = role
	return labels
}
