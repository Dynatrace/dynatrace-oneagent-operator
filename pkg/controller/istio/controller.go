package istio

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	istioclientset "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/clientset/versioned"
	istiov1alpha3 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/istio/v1alpha3"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type probeResult int

const (
	probeObjectFound probeResult = iota
	probeObjectNotFound
	probeTypeFound
	probeTypeNotFound
	probeUnknown
)

// Controller - manager istioclientset and config
type Controller struct {
	istioClient istioclientset.Interface

	logger logr.Logger
	config *rest.Config
}

// NewController - creates new instance of istio controller
func NewController(config *rest.Config) *Controller {
	c := &Controller{
		config: config,
		logger: log.Log.WithName("istio.controller"),
	}
	istioClient, err := c.initialiseIstioClient(config)
	if err != nil {
		return nil
	}
	c.istioClient = istioClient

	return c
}

func (c *Controller) initialiseIstioClient(config *rest.Config) (istioclientset.Interface, error) {
	ic, err := istioclientset.NewForConfig(config)
	if err != nil {
		c.logger.Error(err, fmt.Sprint("istio: failed to initialise client"))
	}

	return ic, err
}

// ReconcileIstio - runs the istio's reconcile workflow,
// creating/deleting VS & SE for external communications
func (c *Controller) ReconcileIstio(oneagent *dynatracev1alpha1.OneAgent,
	dtc dtclient.Client) (updated bool, ok bool) {

	enabled, err := CheckIstioEnabled(c.config)
	if err != nil {
		c.logger.Error(err, "istio: failed to verify Istio availability")
		return false, false
	}
	c.logger.Info("istio: status", "enabled", enabled)

	if !enabled {
		return false, true
	}

	apiHost, err := dtc.GetCommunicationHostForClient()
	if err != nil {
		c.logger.Error(err, "istio: failed to get host for Dynatrace API URL")
		return false, false
	}

	upd, err := c.reconcileIstioConfigurations(oneagent, []dtclient.CommunicationHost{apiHost}, "api-url")
	if err != nil {
		c.logger.Error(err, "istio: error reconciling config for Dynatrace API URL")
		return false, false
	} else if upd {
		return true, true
	}

	// Fetch endpoints via Dynatrace client
	comHosts, err := dtc.GetCommunicationHosts()
	if err != nil {
		c.logger.Error(err, "istio: failed to get Dynatrace communication endpoints")
		return false, false
	}

	if upd, err := c.reconcileIstioConfigurations(oneagent, comHosts, "communication-endpoint"); err != nil {
		c.logger.Error(err, "istio: error reconciling config for Dynatrace communication endpoints")
		return false, false
	} else if upd {
		return true, true
	}

	return false, true
}

func (c *Controller) reconcileIstioConfigurations(instance *dynatracev1alpha1.OneAgent,
	comHosts []dtclient.CommunicationHost, role string) (bool, error) {

	add := c.reconcileIstioCreateConfigurations(instance, comHosts, role)
	rem := c.reconcileIstioRemoveConfigurations(instance, comHosts, role)

	return add || rem, nil
}

func (c *Controller) reconcileIstioRemoveConfigurations(instance *dynatracev1alpha1.OneAgent,
	comHosts []dtclient.CommunicationHost, role string) bool {

	labels := labels.SelectorFromSet(buildIstioLabels(instance.Name, role)).String()
	listOps := &metav1.ListOptions{
		LabelSelector: labels,
	}

	seen := map[string]bool{}
	for _, ch := range comHosts {
		seen[buildNameForEndpoint(instance.Name, ch.Protocol, ch.Host, ch.Port)] = true
	}

	vsUpd := c.removeIstioConfigurationForVirtualService(listOps, seen, instance.Namespace)
	seUpd := c.removeIstioConfigurationForServiceEntry(listOps, seen, instance.Namespace)

	return vsUpd || seUpd
}

func (c *Controller) removeIstioConfigurationForServiceEntry(listOps *metav1.ListOptions,
	seen map[string]bool, namespace string) bool {

	list, err := c.istioClient.NetworkingV1alpha3().ServiceEntries(namespace).List(*listOps)
	if err != nil {
		c.logger.Error(err, fmt.Sprintf("istio: error listing service entries, %v", err))
		return false
	}

	del := false
	for _, se := range list.Items {
		if _, inUse := seen[se.GetName()]; !inUse {
			c.logger.Info(fmt.Sprintf("istio: removing %s: %v", se.Kind, se.GetName()))
			err = c.istioClient.NetworkingV1alpha3().
				ServiceEntries(namespace).
				Delete(se.GetName(), &metav1.DeleteOptions{})
			if err != nil {
				c.logger.Error(err, fmt.Sprintf("istio: error deleteing service entry, %s : %v", se.GetName(), err))
				continue
			}
			del = true
		}
	}

	return del
}

func (c *Controller) removeIstioConfigurationForVirtualService(listOps *metav1.ListOptions,
	seen map[string]bool, namespace string) bool {

	list, err := c.istioClient.NetworkingV1alpha3().VirtualServices(namespace).List(*listOps)
	if err != nil {
		c.logger.Error(err, fmt.Sprintf("istio: error listing virtual service, %v", err))
		return false
	}

	del := false
	for _, vs := range list.Items {
		if _, inUse := seen[vs.GetName()]; !inUse {
			c.logger.Info(fmt.Sprintf("istio: removing %s: %v", vs.Kind, vs.GetName()))
			err = c.istioClient.NetworkingV1alpha3().
				VirtualServices(namespace).
				Delete(vs.GetName(), &metav1.DeleteOptions{})
			if err != nil {
				c.logger.Error(err, fmt.Sprintf("istio: error deleteing virtual service, %s : %v", vs.GetName(), err))
				continue
			}
			del = true
		}
	}

	return del
}

func (c *Controller) reconcileIstioCreateConfigurations(instance *dynatracev1alpha1.OneAgent,
	communicationHosts []dtclient.CommunicationHost, role string) bool {

	crdProbe := c.verifyIstioCrdAvailability(instance)
	if crdProbe != probeTypeFound {
		c.logger.Info("istio: failed to lookup CRD for ServiceEntry/VirtualService: Did you install Istio recently? Please restart the Operator.")
		return false
	}

	configurationUpdated := false
	for _, commHost := range communicationHosts {
		name := buildNameForEndpoint(instance.Name, commHost.Protocol, commHost.Host, commHost.Port)

		createdServiceEntry := c.handleIstioConfigurationForServiceEntry(instance, name, commHost, role)
		createdVirtualService := c.handleIstioConfigurationForVirtualService(instance, name, commHost, role)

		configurationUpdated = configurationUpdated || createdServiceEntry || createdVirtualService
	}

	return configurationUpdated
}

func (c *Controller) verifyIstioCrdAvailability(instance *dynatracev1alpha1.OneAgent) probeResult {
	var probe probeResult

	probe, _ = c.kubernetesObjectProbe(ServiceEntryGVK, instance.Namespace, "")
	if probe == probeTypeNotFound {
		return probe
	}

	probe, _ = c.kubernetesObjectProbe(VirtualServiceGVK, instance.Namespace, "")
	if probe == probeTypeNotFound {
		return probe
	}

	return probeTypeFound
}

func (c *Controller) handleIstioConfigurationForVirtualService(instance *dynatracev1alpha1.OneAgent,
	name string, communicationHost dtclient.CommunicationHost, role string) bool {

	probe, err := c.kubernetesObjectProbe(VirtualServiceGVK, instance.Namespace, name)
	if probe == probeObjectFound {
		return false
	} else if probe == probeUnknown {
		c.logger.Error(err, "istio: failed to query VirtualService")
		return false
	}

	virtualService := buildVirtualService(name, communicationHost.Host, communicationHost.Protocol,
		communicationHost.Port)
	if virtualService == nil {
		return false
	}

	err = c.createIstioConfigurationForVirtualService(instance, virtualService, role)
	if err != nil {
		c.logger.Error(err, "istio: failed to create VirtualService")
		return false
	}
	c.logger.Info("istio: VirtualService created", "objectName", name, "host", communicationHost.Host,
		"port", communicationHost.Port, "protocol", communicationHost.Protocol)

	return true
}

func (c *Controller) handleIstioConfigurationForServiceEntry(instance *dynatracev1alpha1.OneAgent,
	name string, communicationHost dtclient.CommunicationHost, role string) bool {

	probe, err := c.kubernetesObjectProbe(ServiceEntryGVK, instance.Namespace, name)
	if probe == probeObjectFound {
		return false
	} else if probe == probeUnknown {
		c.logger.Error(err, "istio: failed to query ServiceEntry")
		return false
	}

	serviceEntry := buildServiceEntry(name, communicationHost.Host, communicationHost.Protocol, communicationHost.Port)
	err = c.createIstioConfigurationForServiceEntry(instance, serviceEntry, role)
	if err != nil {
		c.logger.Error(err, "istio: failed to create ServiceEntry")
		return false
	}
	c.logger.Info("istio: ServiceEntry created", "objectName", name, "host", communicationHost.Host, "port", communicationHost.Port)

	return true
}

func (c *Controller) createIstioConfigurationForServiceEntry(oneagent *dynatracev1alpha1.OneAgent,
	serviceEntry *istiov1alpha3.ServiceEntry, role string) error {

	serviceEntry.Labels = buildIstioLabels(oneagent.Name, role)
	sve, err := c.istioClient.NetworkingV1alpha3().ServiceEntries(oneagent.Namespace).Create(serviceEntry)
	if err != nil {
		return err
	}
	if sve == nil {
		return fmt.Errorf("Could not create service entry with spec %v", serviceEntry.Spec)
	}

	return nil
}

func (c *Controller) createIstioConfigurationForVirtualService(oneagent *dynatracev1alpha1.OneAgent,
	virtualService *istiov1alpha3.VirtualService, role string) error {

	virtualService.Labels = buildIstioLabels(oneagent.Name, role)
	vs, err := c.istioClient.NetworkingV1alpha3().VirtualServices(oneagent.Namespace).Create(virtualService)
	if err != nil {
		return err
	}
	if vs == nil {
		return fmt.Errorf("Could not create virtual service with spec %v", virtualService.Spec)
	}

	return nil
}

func (c *Controller) kubernetesObjectProbe(gvk schema.GroupVersionKind,
	namespace string, name string) (probeResult, error) {

	var objQuery unstructured.Unstructured
	objQuery.Object = make(map[string]interface{})

	objQuery.SetGroupVersionKind(gvk)

	runtimeClient, err := client.New(c.config, client.Options{})
	if err != nil {
		return probeUnknown, err
	}
	if name == "" {
		err = runtimeClient.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &objQuery)
	} else {
		err = runtimeClient.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, &objQuery)
	}

	return mapErrorToObjectProbeResult(err)
}
