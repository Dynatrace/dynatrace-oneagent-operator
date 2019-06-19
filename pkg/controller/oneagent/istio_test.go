package oneagent

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	testAPIUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
	name       = "dynatrace-oneagent"
	namespace  = "dynatrace"
)

func TestReconcileOneAgent_ReconcileIstio(t *testing.T) {
	oa := newOneAgentSpec()
	oa.ApiUrl = testAPIUrl
	oa.Tokens = "token_test"
	oa.EnableIstio = true
	dynatracev1alpha1.SetDefaults_OneAgentSpec(oa)

	reconcileOA, client := setupReconciler(t, oa)

	// mocking the request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
	_, err := reconcileOA.Reconcile(req)
	if err != nil {
		t.Fatalf("error reconciling: %v", err)
	}
	virtualService := getGVK(client, istio.VirtualServiceGVK)
	if virtualService == nil {
		t.Error("no istio virtual services objects formed")
	}
	serviceEntry := getGVK(client, istio.ServiceEntryGVK)
	if serviceEntry == nil {
		t.Error("no istio objects for service entry")
	}
}

func TestReconcileOneAgent_ReconcileIstioViaDynatraceClient(t *testing.T) {
	oa := newOneAgentSpec()
	oa.ApiUrl = testAPIUrl
	oa.Tokens = "token_test"
	oa.EnableIstio = true
	dynatracev1alpha1.SetDefaults_OneAgentSpec(oa)

	reconcileOA, _ := setupReconciler(t, oa)
	// mocking the request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
	_, err := reconcileOA.Reconcile(req)
	if err != nil {
		t.Fatalf("error reconciling: %v", err)
	}

	// rerun reconcile instio configuration update
	instance := &dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *oa,
	}
	dtc, _ := mockBuildDynatraceClient(instance)
	commHosts, _ := dtc.GetCommunicationHosts()
	commHosts = append(commHosts, dtclient.CommunicationHost{
		Protocol: "https",
		Host:     "https://endpoint3.dev.ruxitlabs.com/communication",
		Port:     443,
	})

	var log = logf.Log.WithName("oneagent.controller.test")

	upd, err := reconcileOA.reconcileIstioConfigurations(log, instance, commHosts, "communication-endpoint")
	if upd != true {
		t.Error("expected true got false, communication endpoints needed to be updated")
	}
}

func getGVK(fake client.Client, gvk schema.GroupVersionKind) *unstructured.UnstructuredList {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)

	listOpts := &client.ListOptions{
		Namespace: "dynatrace",
	}

	fake.List(context.TODO(), listOpts, list)
	return list
}
