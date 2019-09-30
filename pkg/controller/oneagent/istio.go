package oneagent

import (
	"context"
	"fmt"
	"os"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	versionedistioclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/clientset/versioned"
	istiov1alpha3 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/istio/v1alpha3"
	istiohelper "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileOneAgent) reconcileIstio(logger logr.Logger, instance *dynatracev1alpha1.OneAgent, dtc dtclient.Client) (updated bool, ok bool) {
	var err error

	// Determine if cluster runs istio in default cluster
	enabled, err := istiohelper.CheckIstioEnabled(r.config)
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

	add := r.reconcileIstioCreateConfigurations(instance, ic, comHosts, role, logger)
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
		seen[istiohelper.BuildNameForEndpoint(instance.Name, ch.Protocol, ch.Host, ch.Port)] = true
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

	namespace := os.Getenv(k8sutil.WatchNamespaceEnvVar)

	list, err := ic.NetworkingV1alpha3().ServiceEntries(namespace).List(*listOps)
	if err != nil {
		logger.Error(err, fmt.Sprintf("istio: error listing service entries, %v", err))
		return false
	}

	del := false
	for _, se := range list.Items {
		if _, inUse := seen[se.GetName()]; !inUse {
			logger.Info(fmt.Sprintf("istio: removing %s: %v", se.Kind, se.GetName()))
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

	namespace := os.Getenv(k8sutil.WatchNamespaceEnvVar)

	list, err := ic.NetworkingV1alpha3().VirtualServices(namespace).List(*listOps)
	if err != nil {
		logger.Error(err, fmt.Sprintf("istio: error listing virtual service, %v", err))
		return false
	}

	del := false
	for _, vs := range list.Items {
		if _, inUse := seen[vs.GetName()]; !inUse {
			logger.Info(fmt.Sprintf("istio: removing %s: %v", vs.Kind, vs.GetName()))
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

func (r *ReconcileOneAgent) reconcileIstioCreateConfigurations(
	instance *dynatracev1alpha1.OneAgent,
	ic *versionedistioclient.Clientset,
	comHosts []dtclient.CommunicationHost,
	role string, logger logr.Logger) bool {

	created := false

	if _, err := r.configurationNotFound(istiohelper.ServiceEntryGVK, instance.Namespace, ""); meta.IsNoMatchError(err) {
		logger.Info("istio: failed to query ServiceEntry: CRD is not registered. Did you install Istio recently? Please restart the Operator")
		return created
	}

	if _, err := r.configurationNotFound(istiohelper.VirtualServiceGVK, instance.Namespace, ""); meta.IsNoMatchError(err) {
		logger.Info("istio: failed to query VirtualService: CRD is not registered. Did you install Istio recently? Please restart the Operator")
		return created
	}

	for _, ch := range comHosts {
		name := istiohelper.BuildNameForEndpoint(instance.Name, ch.Protocol, ch.Host, ch.Port)

		// Regarding the IsNoMatchError() checks, it's a workaround for,
		// https://github.com/kubernetes-sigs/controller-runtime/issues/321
		//
		// The controller-runtime Client caches CRDs when the process start and doesn't refresh them, so if Istio is
		// installed into Kubernetes after the Operator instance was started, we'll get errors when querying for
		// ServiceEntries, etc.
		//
		// While there is a pending fix for the bug, since this is a minor edge case, we can suggest to the customer to
		// restart the Operator pod.
		if notFound, err := r.configurationNotFound(istiohelper.ServiceEntryGVK, instance.Namespace, name); err != nil {
			logger.Error(err, "istio: failed to query ServiceEntry")
			continue
		} else if notFound {
			logger.Info("istio: creating ServiceEntry", "objectName", name, "host", ch.Host, "port", ch.Port)
			serviceEntry := istiohelper.BuildServiceEntry(name, ch.Host, ch.Protocol, ch.Port)

			if err := r.createIstioConfigurationForServiceEntry(instance, ic, serviceEntry, role, logger); err != nil {
				logger.Error(err, "istio: failed to create ServiceEntry")
				continue
			}
			created = true
		}

		if notFound, err := r.configurationNotFound(istiohelper.VirtualServiceGVK, instance.Namespace, name); err != nil {
			logger.Error(err, "istio: failed to query VirtualService")
			continue
		} else if notFound {
			logger.Info("istio: creating VirtualService", "objectName", name, "host", ch.Host, "port", ch.Port, "protocol", ch.Protocol)
			virtualService := istiohelper.BuildVirtualService(name, ch.Host, ch.Protocol, ch.Port)
			if virtualService == nil {
				continue
			}
			if err := r.createIstioConfigurationForVirtualService(instance, ic, virtualService, role, logger); err != nil {
				logger.Error(err, "istio: failed to create VirtualService")
			}
			created = true
		}
	}

	return created
}

func (r *ReconcileOneAgent) createIstioConfigurationForServiceEntry(
	oneagent *dynatracev1alpha1.OneAgent,
	ic *versionedistioclient.Clientset,
	serviceEntry *istiov1alpha3.ServiceEntry,
	role string, logger logr.Logger) error {

	namespace := os.Getenv(k8sutil.WatchNamespaceEnvVar)
	serviceEntry.Labels = buildIstioLabels(oneagent.Name, role)

	sve, err := ic.NetworkingV1alpha3().ServiceEntries(namespace).Create(serviceEntry)
	if err != nil {
		err = fmt.Errorf("istio: error listing service entries, %v", err)
		logger.Error(err, "istio reconcile")
		return err
	}
	if sve == nil {
		err := fmt.Errorf("Could not create service entry with spec %v", serviceEntry.Spec)
		logger.Error(err, "istio reconcile")
		return err
	}
	return nil
}

func (r *ReconcileOneAgent) createIstioConfigurationForVirtualService(
	oneagent *dynatracev1alpha1.OneAgent,
	ic *versionedistioclient.Clientset,
	virtualService *istiov1alpha3.VirtualService,
	role string, logger logr.Logger) error {

	namespace := os.Getenv(k8sutil.WatchNamespaceEnvVar)
	virtualService.Labels = buildIstioLabels(oneagent.Name, role)

	vs, err := ic.NetworkingV1alpha3().VirtualServices(namespace).Create(virtualService)
	if err != nil {
		err = fmt.Errorf("istio: error listing service entries, %v", err)
		logger.Error(err, "istio reconcile")
		return err
	}
	if vs == nil {
		err := fmt.Errorf("Could not create service entry with spec %v", virtualService.Spec)
		logger.Error(err, "istio reconcile")
		return err
	}
	return nil
}

func (r *ReconcileOneAgent) configurationNotFound(gvk schema.GroupVersionKind, namespace string, name string) (bool, error) {
	var objQuery unstructured.Unstructured
	objQuery.Object = make(map[string]interface{})

	objQuery.SetGroupVersionKind(gvk)

	var err error
	if name == "" {
		err = r.client.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &objQuery)
	} else {
		err = r.client.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, &objQuery)
	}

	if err == nil { // Object found.
		return false, nil
	} else if errors.IsNotFound(err) { // Object not found.
		return true, nil
	}
	return false, err // Other errors
}

func buildIstioLabels(name, role string) map[string]string {
	labels := buildLabels(name)
	labels["dynatrace-istio-role"] = role
	return labels
}
