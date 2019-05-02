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

func (r *ReconcileOneAgent) reconcileIstio(logger logr.Logger, instance *dynatracev1alpha1.OneAgent, dtc dtclient.Client) error {
	var err error

	// Determine if cluster runs istio in default cluster
	enabled, err := istio.CheckIstioEnabled(r.config)
	if err != nil {
		logger.Error(err, "error while checking for Istio availability")
		return err
	}

	logger.Info("Istio status", "enabled", enabled)

	if !enabled {
		return nil
	}

	// Fetch endpoints via Dynatrace client
	comHosts, err := dtc.GetCommunicationHosts()
	if err != nil {
		return err
	}

	err = r.reconcileIstioCreateConfigurations(instance, comHosts, logger)
	if err != nil {
		logger.Error(err, "error reconciling Istio config")
		return err
	}

	r.reconcileIstioRemoveConfigurations(instance, comHosts, logger)

	return nil
}

func (r *ReconcileOneAgent) reconcileIstioRemoveConfigurations(instance *dynatracev1alpha1.OneAgent,
	comHosts []dtclient.CommunicationHost, logger logr.Logger) {

	listOps := &client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: labels.SelectorFromSet(buildLabels(instance.Name)),
	}

	seen := map[string]bool{}
	for _, ch := range comHosts {
		seen[istio.BuildNameForEndpoint(instance.Name, ch.Host, ch.Port)] = true
	}

	r.reconcileIstioRemoveConfiguration(instance, istio.VirtualServiceGVK, listOps, seen, logger)
	r.reconcileIstioRemoveConfiguration(instance, istio.ServiceEntryGVK, listOps, seen, logger)
}

func (r *ReconcileOneAgent) reconcileIstioRemoveConfiguration(instance *dynatracev1alpha1.OneAgent, gvk schema.GroupVersionKind,
	listOps *client.ListOptions, seen map[string]bool, logger logr.Logger) {

	var list unstructured.UnstructuredList
	list.SetGroupVersionKind(gvk)

	if err := r.client.List(context.TODO(), listOps, &list); err != nil {
		return
	}

	for _, item := range list.Items {
		if _, ok := seen[item.GetName()]; !ok {
			logger.Info(fmt.Sprintf("removing Istio %s: %v", gvk.Kind, item.GetName()))
			if err := r.client.Delete(context.TODO(), &item); err != nil {
				logger.Info(fmt.Sprintf("failed to delete Istio %s: %v", gvk.Kind, err))
				continue
			}
		}
	}
}

func (r *ReconcileOneAgent) reconcileIstioCreateConfigurations(instance *dynatracev1alpha1.OneAgent,
	comHosts []dtclient.CommunicationHost, logger logr.Logger) error {

	for _, ch := range comHosts {
		name := istio.BuildNameForEndpoint(instance.Name, ch.Host, ch.Port)

		if notFound := r.configurationExists(istio.ServiceEntryGVK, instance.Namespace, name); notFound {
			logger.Info(fmt.Sprintf("creating Istio ServiceEntry: %s", name))
			payload := istio.BuildServiceEntry(name, ch.Host, ch.Port, ch.Protocol)
			if err := r.reconcileIstioCreateConfiguration(instance, istio.ServiceEntryGVK, payload); err != nil {
				logger.Info(fmt.Sprintf("failed to create Istio ServiceEntry: %v", err))
			}
		}

		if notFound := r.configurationExists(istio.VirtualServiceGVK, instance.Namespace, name); notFound {
			logger.Info(fmt.Sprintf("creating Istio VirtualService: %s", name))
			payload := istio.BuildVirtualService(name, ch.Host, ch.Port, ch.Protocol)
			if err := r.reconcileIstioCreateConfiguration(instance, istio.VirtualServiceGVK, payload); err != nil {
				logger.Info(fmt.Sprintf("failed to create Istio VirtualService: %v", err))
			}
		}
	}

	return nil
}

func (r *ReconcileOneAgent) reconcileIstioCreateConfiguration(instance *dynatracev1alpha1.OneAgent,
	gvk schema.GroupVersionKind, payload []byte) error {

	var obj unstructured.Unstructured
	obj.Object = make(map[string]interface{})

	if err := json.Unmarshal(payload, &obj.Object); err != nil {
		return fmt.Errorf("failed to unmarshal json (%s): %v", payload, err)
	}

	obj.SetGroupVersionKind(gvk)
	obj.SetLabels(buildLabels(instance.Name))

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
