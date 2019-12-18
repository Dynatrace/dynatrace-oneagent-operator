package oneagent

import (
	"context"
	"os"
	"testing"

	apis "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func init() {
	apis.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
	os.Setenv(k8sutil.WatchNamespaceEnvVar, "dynatrace")
}

func TestReconcileOneAgent_ReconcileOnEmptyEnvironmentAndDNSPolicy(t *testing.T) {
	namespace := "dynatrace"
	oaName := "oneagent"

	oaSpec := dynatracev1alpha1.OneAgentSpec{
		ApiUrl:    "https://ENVIRONMENTID.live.dynatrace.com/api",
		DNSPolicy: corev1.DNSClusterFirstWithHostNet,
		Tokens:    oaName,
		Labels: map[string]string{
			"label_key": "label_value",
		},
	}
	dynatracev1alpha1.SetDefaults_OneAgentSpec(&oaSpec)

	fakeClient := fake.NewFakeClient(
		&dynatracev1alpha1.OneAgent{
			ObjectMeta: metav1.ObjectMeta{Name: oaName, Namespace: namespace},
			Spec:       oaSpec,
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: oaName, Namespace: namespace},
		},
	)

	dtClient := &dtclient.MockDynatraceClient{}
	dtClient.On("GetLatestAgentVersion", "unix", "default").Return("42", nil)

	reconciler := &ReconcileOneAgent{
		client:              fakeClient,
		scheme:              scheme.Scheme,
		logger:              logf.ZapLoggerTo(os.Stdout, true),
		dynatraceClientFunc: utils.StaticDynatraceClient(dtClient),
	}

	_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: oaName, Namespace: namespace}})
	assert.NoError(t, err)

	dsActual := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: oaName, Namespace: namespace}, dsActual)
	assert.NoError(t, err, "failed to get DaemonSet")
	assert.Equal(t, namespace, dsActual.Namespace, "wrong namespace")
	assert.Equal(t, oaName, dsActual.GetObjectMeta().GetName(), "wrong name")
	assert.Equal(t, corev1.DNSClusterFirstWithHostNet, dsActual.Spec.Template.Spec.DNSPolicy, "wrong policy")
	mock.AssertExpectationsForObjects(t, dtClient)
}
