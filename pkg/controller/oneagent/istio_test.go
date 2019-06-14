package oneagent

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	testAPIUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
	name       = "dynatrace-oneagent"
	namespace  = "dynatrace"
)

func TestReconcileOneAgent_ReconcileIstio(t *testing.T) {
	reconcileOA, client := setupReconciler(t)

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
	reconcileOA, _ := setupReconciler(t)
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
