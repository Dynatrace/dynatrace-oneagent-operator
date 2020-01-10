package oneagent

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

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
		NewSecret(oaName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
	)

	dtClient := &dtclient.MockDynatraceClient{}
	dtClient.On("GetLatestAgentVersion", "unix", "default").Return("42", nil)
	dtClient.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
	dtClient.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

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

func TestReconcileDynatraceClient_TokenValidation(t *testing.T) {
	namespace := "dynatrace"
	oaName := "oneagent"
	base := dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{Name: oaName, Namespace: namespace},
		Spec: dynatracev1alpha1.OneAgentSpec{
			ApiUrl: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: oaName,
		},
	}
	dynatracev1alpha1.SetDefaults_OneAgentSpec(&base.Spec)

	t.Run("No secret", func(t *testing.T) {
		oa := base.DeepCopy()
		c := fake.NewFakeClient()
		dtcMock := &dtclient.MockDynatraceClient{}

		dtc, ucr, err := reconcileDynatraceClient(oa, c, utils.StaticDynatraceClient(dtcMock), metav1.Now())
		assert.Nil(t, dtc)
		assert.True(t, ucr)
		assert.Error(t, err)

		AssertCondition(t, oa, dynatracev1alpha1.PaaSTokenConditionType, false, dynatracev1alpha1.ReasonTokenSecretNotFound,
			"Secret 'dynatrace:oneagent' not found")
		AssertCondition(t, oa, dynatracev1alpha1.APITokenConditionType, false, dynatracev1alpha1.ReasonTokenSecretNotFound,
			"Secret 'dynatrace:oneagent' not found")

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("PaaS token is empty, API token is missing", func(t *testing.T) {
		oa := base.DeepCopy()
		c := fake.NewFakeClient(NewSecret(oaName, namespace, map[string]string{utils.DynatracePaasToken: ""}))
		dtcMock := &dtclient.MockDynatraceClient{}

		dtc, ucr, err := reconcileDynatraceClient(oa, c, utils.StaticDynatraceClient(dtcMock), metav1.Now())
		assert.Nil(t, dtc)
		assert.True(t, ucr)
		assert.Error(t, err)

		AssertCondition(t, oa, dynatracev1alpha1.PaaSTokenConditionType, false, dynatracev1alpha1.ReasonTokenMissing,
			"Token paasToken on secret dynatrace:oneagent missing")
		AssertCondition(t, oa, dynatracev1alpha1.APITokenConditionType, false, dynatracev1alpha1.ReasonTokenMissing,
			"Token apiToken on secret dynatrace:oneagent missing")

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("Unauthorized PaaS token, unexpected error for API token request", func(t *testing.T) {
		oa := base.DeepCopy()
		c := fake.NewFakeClient(NewSecret(oaName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes(nil), dtclient.ServerError{Code: 401, Message: "Token Authentication failed"})
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes(nil), fmt.Errorf("random error"))

		dtc, ucr, err := reconcileDynatraceClient(oa, c, utils.StaticDynatraceClient(dtcMock), metav1.Now())
		assert.Equal(t, dtcMock, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)

		AssertCondition(t, oa, dynatracev1alpha1.PaaSTokenConditionType, false, dynatracev1alpha1.ReasonTokenUnauthorized,
			"Token on secret dynatrace:oneagent unauthorized")
		AssertCondition(t, oa, dynatracev1alpha1.APITokenConditionType, false, dynatracev1alpha1.ReasonTokenError,
			"error when querying token on secret dynatrace:oneagent: random error")

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("PaaS token has wrong scope, API token is ready", func(t *testing.T) {
		oa := base.DeepCopy()
		c := fake.NewFakeClient(NewSecret(oaName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		dtc, ucr, err := reconcileDynatraceClient(oa, c, utils.StaticDynatraceClient(dtcMock), metav1.Now())
		assert.Equal(t, dtcMock, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)

		AssertCondition(t, oa, dynatracev1alpha1.PaaSTokenConditionType, false, dynatracev1alpha1.ReasonTokenScopeMissing,
			"Token on secret dynatrace:oneagent missing scope InstallerDownload")
		AssertCondition(t, oa, dynatracev1alpha1.APITokenConditionType, true, dynatracev1alpha1.ReasonTokenReady, "Ready")

		mock.AssertExpectationsForObjects(t, dtcMock)
	})
}

func TestReconcileDynatraceClient_ProbeRequests(t *testing.T) {
	now := metav1.Now()

	namespace := "dynatrace"
	oaName := "oneagent"
	base := dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{Name: oaName, Namespace: namespace},
		Spec: dynatracev1alpha1.OneAgentSpec{
			ApiUrl: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: oaName,
		},
	}
	dynatracev1alpha1.SetDefaults_OneAgentSpec(&base.Spec)
	base.SetCondition(dynatracev1alpha1.APITokenConditionType, corev1.ConditionTrue, dynatracev1alpha1.ReasonTokenReady, "Ready")
	base.SetCondition(dynatracev1alpha1.PaaSTokenConditionType, corev1.ConditionTrue, dynatracev1alpha1.ReasonTokenReady, "Ready")

	c := fake.NewFakeClient(NewSecret(oaName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}))

	t.Run("No request if last probe was recent", func(t *testing.T) {
		lastAPIProbe := metav1.NewTime(now.Add(-3 * time.Minute))
		lastPaaSProbe := metav1.NewTime(now.Add(-3 * time.Minute))

		oa := base.DeepCopy()
		oa.Status.LastAPITokenProbeTimestamp = &lastAPIProbe
		oa.Status.LastPaaSTokenProbeTimestamp = &lastPaaSProbe

		dtcMock := &dtclient.MockDynatraceClient{}

		dtc, ucr, err := reconcileDynatraceClient(oa, c, utils.StaticDynatraceClient(dtcMock), now)
		assert.Equal(t, dtcMock, dtc)
		assert.False(t, ucr)
		assert.NoError(t, err)
		if assert.NotNil(t, oa.Status.LastAPITokenProbeTimestamp) {
			assert.Equal(t, *oa.Status.LastAPITokenProbeTimestamp, lastAPIProbe)
		}
		if assert.NotNil(t, oa.Status.LastPaaSTokenProbeTimestamp) {
			assert.Equal(t, *oa.Status.LastPaaSTokenProbeTimestamp, lastPaaSProbe)
		}
		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("Make request if last probe was not recent", func(t *testing.T) {
		lastAPIProbe := metav1.NewTime(now.Add(-10 * time.Minute))
		lastPaaSProbe := metav1.NewTime(now.Add(-10 * time.Minute))

		oa := base.DeepCopy()
		oa.Status.LastAPITokenProbeTimestamp = &lastAPIProbe
		oa.Status.LastPaaSTokenProbeTimestamp = &lastPaaSProbe

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		dtc, ucr, err := reconcileDynatraceClient(oa, c, utils.StaticDynatraceClient(dtcMock), now)
		assert.Equal(t, dtcMock, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)
		if assert.NotNil(t, oa.Status.LastAPITokenProbeTimestamp) {
			assert.Equal(t, *oa.Status.LastAPITokenProbeTimestamp, now)
		}
		if assert.NotNil(t, oa.Status.LastPaaSTokenProbeTimestamp) {
			assert.Equal(t, *oa.Status.LastPaaSTokenProbeTimestamp, now)
		}
		mock.AssertExpectationsForObjects(t, dtcMock)
	})
}

func AssertCondition(t *testing.T, oa *dynatracev1alpha1.OneAgent, ct dynatracev1alpha1.OneAgentConditionType, status bool, reason string, message string) {
	t.Helper()
	s := corev1.ConditionFalse
	if status {
		s = corev1.ConditionTrue
	}
	cond := oa.Condition(ct)
	assert.Equal(t, s, cond.Status)
	assert.Equal(t, reason, cond.Reason)
	assert.Equal(t, message, cond.Message)
}

func NewSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}
