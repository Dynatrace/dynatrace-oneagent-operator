package oneagent

import (
	"context"
	"encoding/json"
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
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

	upd, err := r.reconcileIstioConfigurations(logger, instance, []dtclient.CommunicationHost{apiHost}, "api-url")
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

	if upd, err := r.reconcileIstioConfigurations(logger, instance, comHosts, "communication-endpoint"); err != nil {
		logger.Error(err, "istio: error reconciling config for Dynatrace communication endpoints")
		return false, false
	} else if upd {
		return true, true
	}

	return false, true
}

func (r *ReconcileOneAgent) reconcileIstioConfigurations(logger logr.Logger, instance *dynatracev1alpha1.OneAgent,
	comHosts []dtclient.CommunicationHost, role string) (bool, error) {
	add := r.reconcileIstioCreateConfigurations(instance, comHosts, role, logger)
	rem := r.reconcileIstioRemoveConfigurations(instance, comHosts, role, logger)
	return add || rem, nil
}

func (r *ReconcileOneAgent) reconcileIstioRemoveConfigurations(instance *dynatracev1alpha1.OneAgent,
	comHosts []dtclient.CommunicationHost, role string, logger logr.Logger) bool {

	listOps := &client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: labels.SelectorFromSet(buildIstioLabels(instance.Name, role)),
	}

	seen := map[string]bool{}
	for _, ch := range comHosts {
		seen[istio.BuildNameForEndpoint(instance.Name, ch.Host, ch.Port)] = true
	}

	vsUpd := r.reconcileIstioRemoveConfiguration(instance, istio.VirtualServiceGVK, listOps, seen, logger)
	seUpd := r.reconcileIstioRemoveConfiguration(instance, istio.ServiceEntryGVK, listOps, seen, logger)

	return vsUpd || seUpd
}

func (r *ReconcileOneAgent) reconcileIstioRemoveConfiguration(instance *dynatracev1alpha1.OneAgent, gvk schema.GroupVersionKind,
	listOps *client.ListOptions, seen map[string]bool, logger logr.Logger) bool {

	var list unstructured.UnstructuredList
	list.SetGroupVersionKind(gvk)

	if err := r.client.List(context.TODO(), listOps, &list); err != nil {
		logger.Error(err, fmt.Sprintf("istio: failed to list %s objects", gvk.Kind))
		return false
	}

	del := false

	for _, item := range list.Items {
		if _, inUse := seen[item.GetName()]; !inUse {
			logger.Info(fmt.Sprintf("istio: removing %s: %v", gvk.Kind, item.GetName()))
			if err := r.client.Delete(context.TODO(), &item); err != nil {
				logger.Error(err, fmt.Sprintf("istio: failed to delete %s", gvk.Kind))
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
