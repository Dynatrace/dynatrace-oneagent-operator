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
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileOneAgent) reconcileIstio(logger logr.Logger, instance *dynatracev1alpha1.OneAgent, dtc dtclient.Client) error {
	var err error

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}
	// determine if cluster runs istio in default cluster
	err = istio.CheckIstioService(cfg)
	if err != nil {
		log.Error(err, "error checking for istio")
		return err
	}

	// fetch endpoints via dynatrace client
	communicationHosts, err := dtc.GetCommunicationHosts()
	if err != nil {
		return err
	}

	err = r.reconcileIstioCreateConfigurations(instance, communicationHosts, logger)
	if err != nil {
		return err
	}

	r.reconileIstioRemoveConfigurations(instance, communicationHosts, logger)

	return nil
}

func (r *ReconcileOneAgent) reconileIstioRemoveConfigurations(instance *dynatracev1alpha1.OneAgent, communicationHosts []dtclient.CommunicationHost, logger logr.Logger) {
	listOps := &client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: labels.SelectorFromSet(buildLabels(instance.Name)),
	}

	seen := map[string]bool{}
	for _, ch := range communicationHosts {
		seen[istio.BuildNameForEndpoint(instance.Name, ch.Host, ch.Port)] = true
	}

	r.reconileIstioRemoveConfiguration(instance, istio.VirtualServiceGVK, listOps, seen, logger)
	r.reconileIstioRemoveConfiguration(instance, istio.ServiceEntryGVK, listOps, seen, logger)
}

func (r *ReconcileOneAgent) reconileIstioRemoveConfiguration(instance *dynatracev1alpha1.OneAgent, gvk schema.GroupVersionKind, listOps *client.ListOptions, seen map[string]bool, logger logr.Logger) {
	var err error
	var list unstructured.UnstructuredList

	list.SetGroupVersionKind(gvk)

	err = r.client.List(context.TODO(), listOps, &list)
	if err != nil {
		return
	}

	for _, item := range list.Items {
		if _, ok := seen[item.GetName()]; !ok {
			logger.Info(fmt.Sprintf("removing istio %s: %v", gvk.Kind, item.GetName()))
			err = r.client.Delete(context.TODO(), &item)
			if err != nil {
				logger.Info(fmt.Sprintf("failed to delete istio %s: %v", gvk.Kind, err))
				continue
			}
		}
	}
}

func (r *ReconcileOneAgent) reconcileIstioCreateConfigurations(instance *dynatracev1alpha1.OneAgent, communicationHosts []dtclient.CommunicationHost, logger logr.Logger) error {
	for _, ch := range communicationHosts {
		name := istio.BuildNameForEndpoint(instance.Name, ch.Host, ch.Port)

		if notFound := r.configurationExists(istio.ServiceEntryGVK, instance.Namespace, name); notFound {
			logger.Info(fmt.Sprintf("creating istio serviceentry: %s", name))
			payload := istio.BuildServiceEntry(name, ch.Host, ch.Port, ch.Protocol)
			if err := r.reconcileIstioCreateConfiguration(instance, istio.ServiceEntryGVK, payload); err != nil {
				logger.Info(fmt.Sprintf("failed to create istio serviceentry: %v", err))
			}
		}

		if notFound := r.configurationExists(istio.VirtualServiceGVK, instance.Namespace, name); notFound {
			logger.Info(fmt.Sprintf("creating istio virtualservice: %s", name))
			payload := istio.BuildVirtualService(name, ch.Host, ch.Port, ch.Protocol)
			if err := r.reconcileIstioCreateConfiguration(instance, istio.VirtualServiceGVK, payload); err != nil {
				logger.Info(fmt.Sprintf("failed to create istio virtualservice: %v", err))
			}
		}
	}

	return nil
}

func (r *ReconcileOneAgent) reconcileIstioCreateConfiguration(instance *dynatracev1alpha1.OneAgent, gvk schema.GroupVersionKind, payload []byte) error {
	var err error
	var obj unstructured.Unstructured
	obj.Object = make(map[string]interface{})

	err = json.Unmarshal(payload, &obj.Object)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json (%s): %v", payload, err)
	}

	obj.SetGroupVersionKind(gvk)
	obj.SetLabels(buildLabels(instance.Name))
	err = controllerutil.SetControllerReference(instance, &obj, r.scheme)
	if err != nil {
		return fmt.Errorf("failed to set owner reference: %v", err)
	}

	err = r.client.Create(context.TODO(), &obj)
	if err != nil {
		return fmt.Errorf("failed to create istio configuration: %v", err)
	}

	return nil
}

func (r *ReconcileOneAgent) configurationExists(gvk schema.GroupVersionKind, namespace string, name string) bool {
	var objQuery unstructured.Unstructured
	objQuery.Object = make(map[string]interface{})

	objQuery.SetGroupVersionKind(gvk)
	key := client.ObjectKey{Namespace: namespace, Name: name}

	err := r.client.Get(context.TODO(), key, &objQuery)

	return errors.IsNotFound(err)
}
