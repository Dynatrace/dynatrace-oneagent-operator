package oneagent

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileOneAgent_ReconcileOnEmptyEnvironment(t *testing.T) {
	var (
		name      = "dynatrace-oneagent"
		namespace = "dynatrace"
	)

	oa := newOneAgentSpec()
	oa.ApiUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
	oa.Tokens = "token_test"
	dynatracev1alpha1.SetDefaults_OneAgentSpec(oa)

	instance := &dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *oa,
	}

	scheme := scheme.Scheme
	scheme.AddKnownTypes(dynatracev1alpha1.SchemeGroupVersion, instance)

	client := fake.NewFakeClient(instance)
	// reconcile oneagent
	reconcileOA := &ReconcileOneAgent{client: client, scheme: scheme}

	// mocking the request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
	_, err := reconcileOA.Reconcile(req)
	if err != nil {
		t.Fatalf("error reconciling : %v", err)
	}

	// Check if deamonset has been created and has correct namespace and name.
	dsActual := &appsv1.DaemonSet{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, dsActual)
	if err != nil {
		t.Fatalf("get deamonset: (%v)", err)
	}
	if dsActual.Namespace != namespace {
		t.Errorf("wrong namespace, expected %v, got %v", namespace, dsActual.Namespace)
	}
	if dsActual.GetObjectMeta().GetName() != name {
		t.Errorf("wrong name, expected %v, got %v", name, dsActual.GetObjectMeta().GetName())
	}
}
