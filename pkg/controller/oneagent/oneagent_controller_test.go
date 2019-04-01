package oneagent

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestOneAgentController(t *testing.T) {

	var (
		name      = "dynatrace-oneagent"
		namespace = "dynatrace"
	)

	oa := newOneAgentSpec()
	dynatracev1alpha1.SetDefaults_OneAgentSpec(oa)
	if oa.ApiUrl == "" {
		oa.ApiUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
	}
	oa.Tokens = "token123"
	ds := newDaemonSetSpec()
	ds.Template.Spec.Containers = []corev1.Container{{
		Image:     "docker.io/dynatrace/oneagent",
		Args:      []string{"INFRO_ONLY=1"},
		Resources: newResourceRequirements(),
		Env:       newEnvVar(),
	}}
	ds.Template.Spec.Tolerations = []corev1.Toleration{}
	ds.Template.Spec.NodeSelector = map[string]string{"k": "v"}
	ds.Template.Spec.PriorityClassName = "class"

	copyDaemonSetSpecToOneAgentSpec(ds, oa)

	instance := &dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *oa,
	}
	objs := []runtime.Object{
		instance,
	}

	scheme := scheme.Scheme
	scheme.AddKnownTypes(dynatracev1alpha1.SchemeGroupVersion, instance)

	client := fake.NewFakeClient(objs...)
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

	// Check if deamonset has been created and has the correct size.
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
