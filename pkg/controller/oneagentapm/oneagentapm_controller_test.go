package oneagentapm

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

const (
	apiURL    = "https://ENVIRONMENTID.live.dynatrace.com/api"
	name      = "oneagent"
	namespace = "dynatrace"
)

func TestReconcileOneAgentAPM(t *testing.T) {
	fakeClient := fake.NewFakeClient(
		&dynatracev1alpha1.OneAgentAPM{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: dynatracev1alpha1.OneAgentAPMSpec{
				BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
					APIURL: apiURL,
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Data:       map[string][]byte{utils.DynatracePaasToken: []byte("42")},
		},
	)

	dtClient := &dtclient.MockDynatraceClient{}
	dtClient.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)

	reconciler := &ReconcileOneAgentAPM{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		logger:    logf.ZapLoggerTo(os.Stdout, true),
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              fakeClient,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtClient),
			UpdatePaaSToken:     true,
		},
	}

	_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}})
	assert.NoError(t, err)

	var result dynatracev1alpha1.OneAgentAPM
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &result))
	assert.Equal(t, namespace, result.GetNamespace())
	assert.Equal(t, name, result.GetName())
	assert.True(t, result.Status.Conditions.IsTrueFor(dynatracev1alpha1.PaaSTokenConditionType))
	assert.True(t, result.Status.Conditions.IsUnknownFor(dynatracev1alpha1.APITokenConditionType))
	mock.AssertExpectationsForObjects(t, dtClient)
}

func TestReconcileOneAgentAPM_MissingToken(t *testing.T) {
	fakeClient := fake.NewFakeClient(
		&dynatracev1alpha1.OneAgentAPM{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: dynatracev1alpha1.OneAgentAPMSpec{
				BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
					APIURL: apiURL,
				},
			},
		},
	)

	dtClient := &dtclient.MockDynatraceClient{}

	reconciler := &ReconcileOneAgentAPM{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		logger:    logf.ZapLoggerTo(os.Stdout, true),
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              fakeClient,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtClient),
			UpdatePaaSToken:     true,
		},
	}

	_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}})
	assert.NotNil(t, err)
	assert.Equal(t, "Secret 'dynatrace:oneagent' not found", err.Error())

	var result dynatracev1alpha1.OneAgentAPM
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &result))
	assert.Equal(t, namespace, result.GetNamespace())
	assert.Equal(t, name, result.GetName())
	assert.True(t, result.Status.Conditions.IsFalseFor(dynatracev1alpha1.PaaSTokenConditionType))
	assert.True(t, result.Status.Conditions.IsUnknownFor(dynatracev1alpha1.APITokenConditionType))
	mock.AssertExpectationsForObjects(t, dtClient)
}
