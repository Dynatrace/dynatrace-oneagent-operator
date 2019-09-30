package oneagent

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	versionedistioclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/clientset/versioned"
	istiov1alpha3 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/istio/v1alpha3"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	istiohelper "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProbeResult int

const (
	probeObjectFound ProbeResult = iota
	probeObjectNotFound
	probeTypeFound
	probeTypeNotFound
	probeUnknown
)

func (r *ReconcileOneAgent) reconcileIstio(
	logger logr.Logger,
	instance *dynatracev1alpha1.OneAgent,
	dtc dtclient.Client,
) (updated bool, ok bool) {

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

	apiHost, err := dtc.GetCommunicationHostForClient()
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
	logger logr.Logger,
) (bool, error) {

	add := r.reconcileIstioCreateConfigurations(instance, ic, comHosts, role, logger)
	rem := r.reconcileIstioRemoveConfigurations(instance, ic, comHosts, role, logger)
	return add || rem, nil
}

func (r *ReconcileOneAgent) reconcileIstioRemoveConfigurations(
	instance *dynatracev1alpha1.OneAgent,
	ic *versionedistioclient.Clientset,
	comHosts []dtclient.CommunicationHost,
	role string,
	logger logr.Logger,
) bool {

	labels := labels.SelectorFromSet(buildIstioLabels(instance.Name, role)).String()
	listOps := &metav1.ListOptions{
		LabelSelector: labels,
	}

	seen := map[string]bool{}
	for _, ch := range comHosts {
		seen[istiohelper.BuildNameForEndpoint(instance.Name, ch.Protocol, ch.Host, ch.Port)] = true
	}

	vsUpd := r.removeIstioConfigurationForVirtualService(ic, listOps, seen, instance.Namespace, logger)
	seUpd := r.removeIstioConfigurationForServiceEntry(ic, listOps, seen, instance.Namespace, logger)

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
	namespace string,
	logger logr.Logger,
) bool {

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
	namespace string,
	logger logr.Logger,
) bool {

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
	istioClient *versionedistioclient.Clientset,
	communicationHosts []dtclient.CommunicationHost,
	role string,
	logger logr.Logger,
) bool {

	crdProbe := r.verifyIstioCrdAvailability(instance, logger)
	if crdProbe != probeTypeFound {
		logger.Info("istio: failed to lookup CRD for ServiceEntry/VirtualService: Did you install Istio recently? Please restart the Operator.")
		return false
	}

	configurationUpdated := false
	for _, ch := range communicationHosts {
		name := istiohelper.BuildNameForEndpoint(instance.Name, ch.Protocol, ch.Host, ch.Port)

		createdServiceEntry := r.handleIstioConfigurationForServiceEntry(instance, name, logger, ch, istioClient, role)
		createdVirtualService := r.handleIstioConfigurationForVirtualService(instance, name, logger, ch, istioClient, role)

		configurationUpdated = configurationUpdated || createdServiceEntry || createdVirtualService
	}

	return configurationUpdated
}

func (r *ReconcileOneAgent) verifyIstioCrdAvailability(
	instance *dynatracev1alpha1.OneAgent,
	logger logr.Logger,
) ProbeResult {

	var probe ProbeResult

	probe, _ = r.kubernetesObjectProbe(istio.ServiceEntryGVK, instance.Namespace, "")
	if probe == probeTypeNotFound {
		return probe
	}

	probe, _ = r.kubernetesObjectProbe(istio.VirtualServiceGVK, instance.Namespace, "")
	if probe == probeTypeNotFound {
		return probe
	}

	return probeTypeFound
}

func (r *ReconcileOneAgent) handleIstioConfigurationForVirtualService(
	instance *dynatracev1alpha1.OneAgent,
	name string,
	logger logr.Logger,
	communicationHost dtclient.CommunicationHost,
	istioClient *versionedistioclient.Clientset,
	role string,
) bool {

	probe, err := r.kubernetesObjectProbe(istio.VirtualServiceGVK, instance.Namespace, name)
	if probe == probeObjectFound {
		return false
	} else if probe == probeUnknown {
		logger.Error(err, "istio: failed to query VirtualService")
		return false
	}

	virtualService := istio.BuildVirtualService(name, communicationHost.Host, communicationHost.Protocol, communicationHost.Port)
	if virtualService == nil {
		return false
	}

	err = r.createIstioConfigurationForVirtualService(instance, istioClient, virtualService, role, logger)
	if err != nil {
		logger.Error(err, "istio: failed to create VirtualService")
		return false
	}
	logger.Info("istio: VirtualService created", "objectName", name, "host", communicationHost.Host, "port", communicationHost.Port, "protocol", communicationHost.Protocol)

	return true
}

func (r *ReconcileOneAgent) handleIstioConfigurationForServiceEntry(
	instance *dynatracev1alpha1.OneAgent,
	name string,
	logger logr.Logger,
	communicationHost dtclient.CommunicationHost,
	istioClient *versionedistioclient.Clientset,
	role string,
) bool {

	probe, err := r.kubernetesObjectProbe(istio.ServiceEntryGVK, instance.Namespace, name)
	if probe == probeObjectFound {
		return false
	} else if probe == probeUnknown {
		logger.Error(err, "istio: failed to query ServiceEntry")
		return false
	}

	serviceEntry := istiohelper.BuildServiceEntry(name, communicationHost.Host, communicationHost.Protocol, communicationHost.Port)
	err = r.createIstioConfigurationForServiceEntry(instance, istioClient, serviceEntry, role, logger)
	if err != nil {
		logger.Error(err, "istio: failed to create ServiceEntry")
		return false
	}
	logger.Info("istio: ServiceEntry created", "objectName", name, "host", communicationHost.Host, "port", communicationHost.Port)

	return true
}

func (r *ReconcileOneAgent) createIstioConfigurationForServiceEntry(
	oneagent *dynatracev1alpha1.OneAgent,
	ic *versionedistioclient.Clientset,
	serviceEntry *istiov1alpha3.ServiceEntry,
	role string,
	logger logr.Logger,
) error {

	serviceEntry.Labels = buildIstioLabels(oneagent.Name, role)
	sve, err := ic.NetworkingV1alpha3().ServiceEntries(oneagent.Namespace).Create(serviceEntry)
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
	role string,
	logger logr.Logger,
) error {

	virtualService.Labels = buildIstioLabels(oneagent.Name, role)
	vs, err := ic.NetworkingV1alpha3().VirtualServices(oneagent.Namespace).Create(virtualService)
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

func (r *ReconcileOneAgent) kubernetesObjectProbe(
	gvk schema.GroupVersionKind,
	namespace string,
	name string,
) (ProbeResult, error) {

	var objQuery unstructured.Unstructured
	objQuery.Object = make(map[string]interface{})

	objQuery.SetGroupVersionKind(gvk)

	var err error
	if name == "" {
		err = r.client.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &objQuery)
	} else {
		err = r.client.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, &objQuery)
	}

	return mapErrorToObjectProbeResult(err)
}

func mapErrorToObjectProbeResult(err error) (ProbeResult, error) {
	if err != nil {
		if errors.IsNotFound(err) {
			return probeObjectNotFound, err
		} else if meta.IsNoMatchError(err) {
			return probeTypeNotFound, err
		}

		return probeUnknown, err
	}

	return probeObjectFound, nil
}

func buildIstioLabels(name, role string) map[string]string {
	labels := buildLabels(name)
	labels["dynatrace-istio-role"] = role
	return labels
}
