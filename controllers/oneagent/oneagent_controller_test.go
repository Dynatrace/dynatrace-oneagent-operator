package oneagent

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/dtclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	utilruntime.Must(dynatracev1alpha1.AddToScheme(scheme.Scheme))
}

var consoleLogger = zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout))

var sampleKubeSystemNS = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "kube-system",
		UID:  "01234-5678-9012-3456",
	},
}

func TestReconcileOneAgent_ReconcileOnEmptyEnvironmentAndDNSPolicy(t *testing.T) {
	namespace := "dynatrace"
	oaName := "oneagent"

	oaSpec := dynatracev1alpha1.OneAgentSpec{
		BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: oaName,
		},
		DNSPolicy: corev1.DNSClusterFirstWithHostNet,
		Labels: map[string]string{
			"label_key": "label_value",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			&dynatracev1alpha1.OneAgent{
				ObjectMeta: metav1.ObjectMeta{Name: oaName, Namespace: namespace},
				Spec:       oaSpec,
			},
			NewSecret(oaName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
			sampleKubeSystemNS,
		).
		Build()

	dtClient := &dtclient.MockDynatraceClient{}
	dtClient.On("GetLatestAgentVersion", "unix", "default").Return("42", nil)
	dtClient.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
	dtClient.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)
	dtClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{TenantUUID: "abc123456"}, nil)

	reconciler := &ReconcileOneAgent{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              fakeClient,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtClient),
			UpdatePaaSToken:     true,
			UpdateAPIToken:      true,
		},
	}

	_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: oaName, Namespace: namespace}})
	assert.NoError(t, err)

	dsActual := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: oaName, Namespace: namespace}, dsActual)
	assert.NoError(t, err, "failed to get DaemonSet")
	assert.Equal(t, namespace, dsActual.Namespace, "wrong namespace")
	assert.Equal(t, oaName, dsActual.GetObjectMeta().GetName(), "wrong name")
	assert.Equal(t, corev1.DNSClusterFirstWithHostNet, dsActual.Spec.Template.Spec.DNSPolicy, "wrong policy")
	mock.AssertExpectationsForObjects(t, dtClient)
}

func TestReconcile_PhaseSetCorrectly(t *testing.T) {
	namespace := "dynatrace"
	oaName := "oneagent"
	base := dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{Name: oaName, Namespace: namespace},
		Spec: dynatracev1alpha1.OneAgentSpec{
			BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
				APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
				Tokens: oaName,
			},
		},
	}
	meta.SetStatusCondition(&base.Status.Conditions, metav1.Condition{
		Type:    dynatracev1alpha1.APITokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1alpha1.ReasonTokenReady,
		Message: "Ready",
	})
	meta.SetStatusCondition(&base.Status.Conditions, metav1.Condition{
		Type:    dynatracev1alpha1.PaaSTokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1alpha1.ReasonTokenReady,
		Message: "Ready",
	})

	t.Run("SetPhaseOnError called with different values, object and return value correctly modified", func(t *testing.T) {
		oa := base.DeepCopy()

		res := oa.GetOneAgentStatus().SetPhaseOnError(nil)
		assert.False(t, res)
		assert.Equal(t, oa.Status.Phase, dynatracev1alpha1.OneAgentPhaseType(""))

		res = oa.GetOneAgentStatus().SetPhaseOnError(errors.New("dummy error"))
		assert.True(t, res)

		if assert.NotNil(t, oa.Status.Phase) {
			assert.Equal(t, dynatracev1alpha1.Error, oa.Status.Phase)
		}
	})

	// arrange
	c := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			NewSecret(oaName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
			sampleKubeSystemNS,
		).
		Build()
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtcMock),
			UpdatePaaSToken:     true,
			UpdateAPIToken:      true,
		},
	}

	t.Run("reconcileRollout Phase is set to deploying, if agent version is not set on OneAgent object", func(t *testing.T) {
		// arrange
		oa := base.DeepCopy()
		oa.Status.Version = ""

		// act
		updateCR, err := reconciler.reconcileRollout(context.TODO(), consoleLogger, oa, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, err, nil)
		assert.Equal(t, dynatracev1alpha1.Deploying, oa.Status.Phase)
		assert.Equal(t, version, oa.Status.Version)
	})

	t.Run("reconcileRollout Phase not changing, if agent version is already set on OneAgent object", func(t *testing.T) {
		// arrange
		oa := base.DeepCopy()
		oa.Status.Version = version
		oa.Status.Tokens = utils.GetTokensName(oa)

		// act
		updateCR, err := reconciler.reconcileRollout(context.TODO(), consoleLogger, oa, dtcMock)

		// assert
		assert.False(t, updateCR)
		assert.Equal(t, nil, err)
		assert.Equal(t, dynatracev1alpha1.OneAgentPhaseType(""), oa.Status.Phase)
	})

	t.Run("reconcileVersion Phase not changing", func(t *testing.T) {
		// arrange
		oa := base.DeepCopy()
		oa.Status.Version = version

		// act
		_, err := reconciler.reconcileVersion(context.TODO(), consoleLogger, oa, dtcMock)

		// assert
		assert.Equal(t, nil, err)
		assert.Equal(t, dynatracev1alpha1.OneAgentPhaseType(""), oa.Status.Phase)
	})
}

func TestReconcile_TokensSetCorrectly(t *testing.T) {
	namespace := "dynatrace"
	oaName := "oneagent"
	base := dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{Name: oaName, Namespace: namespace},
		Spec: dynatracev1alpha1.OneAgentSpec{
			BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
				APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
				Tokens: oaName,
			},
		},
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			NewSecret(oaName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
			sampleKubeSystemNS).
		Build()
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtcMock),
			UpdatePaaSToken:     true,
			UpdateAPIToken:      true,
		},
	}

	t.Run("reconcileRollout Tokens status set, if empty", func(t *testing.T) {
		// arrange
		oa := base.DeepCopy()
		oa.Spec.Tokens = ""
		oa.Status.Tokens = ""

		// act
		updateCR, err := reconciler.reconcileRollout(context.TODO(), consoleLogger, oa, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, utils.GetTokensName(oa), oa.Status.Tokens)
		assert.Equal(t, nil, err)
	})
	t.Run("reconcileRollout Tokens status set, if status has wrong name", func(t *testing.T) {
		// arrange
		oa := base.DeepCopy()
		oa.Spec.Tokens = ""
		oa.Status.Tokens = "not the actual name"

		// act
		updateCR, err := reconciler.reconcileRollout(context.TODO(), consoleLogger, oa, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, utils.GetTokensName(oa), oa.Status.Tokens)
		assert.Equal(t, nil, err)
	})

	t.Run("reconcileRollout Tokens status set, not equal to defined name", func(t *testing.T) {
		c = fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				NewSecret(oaName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
				sampleKubeSystemNS).
			Build()

		reconciler := &ReconcileOneAgent{
			client:    c,
			apiReader: c,
			scheme:    scheme.Scheme,
			logger:    consoleLogger,
			dtcReconciler: &utils.DynatraceClientReconciler{
				Client:              c,
				DynatraceClientFunc: utils.StaticDynatraceClient(dtcMock),
				UpdatePaaSToken:     true,
				UpdateAPIToken:      true,
			},
		}

		// arrange
		customTokenName := "custom-token-name"
		oa := base.DeepCopy()
		oa.Status.Tokens = utils.GetTokensName(oa)
		oa.Spec.Tokens = customTokenName

		// act
		updateCR, err := reconciler.reconcileRollout(context.TODO(), consoleLogger, oa, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, utils.GetTokensName(oa), oa.Status.Tokens)
		assert.Equal(t, customTokenName, oa.Status.Tokens)
		assert.Equal(t, nil, err)
	})
}

func TestReconcile_InstancesSet(t *testing.T) {
	namespace := "dynatrace"
	oaName := "oneagent"
	base := dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{Name: oaName, Namespace: namespace},
		Spec: dynatracev1alpha1.OneAgentSpec{
			BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
				APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
				Tokens: oaName,
			},
		},
	}

	// arrange
	c := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			NewSecret(oaName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
			sampleKubeSystemNS).
		Build()
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	oldVersion := "1.186"
	hostIP := "1.2.3.4"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)
	dtcMock.On("GetAgentVersionForIP", hostIP).Return(version, nil)
	dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{utils.DynatracePaasToken}, nil)
	dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{utils.DynatraceApiToken}, nil)

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtcMock),
			UpdatePaaSToken:     true,
			UpdateAPIToken:      true,
		},
	}

	t.Run("reconcileImpl Instances set, if agentUpdateDisabled is false", func(t *testing.T) {
		oa := base.DeepCopy()
		oa.Spec.DisableAgentUpdate = false
		oa.Status.Version = oldVersion
		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-enabled"
		pod.Namespace = namespace
		pod.Labels = buildLabels(oaName)
		pod.Spec = newPodSpecForCR(oa, false, consoleLogger, "cluster1")
		pod.Status.HostIP = hostIP
		oa.Status.Tokens = utils.GetTokensName(oa)

		rec := reconciliation{log: consoleLogger, instance: oa, requeueAfter: 30 * time.Minute}
		err := reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		reconciler.reconcileImpl(context.TODO(), &rec)

		assert.NotNil(t, oa.Status.Instances)
		assert.NotEmpty(t, oa.Status.Instances)
	})

	t.Run("reconcileImpl Instances set, if agentUpdateDisabled is true", func(t *testing.T) {
		oa := base.DeepCopy()
		oa.Spec.DisableAgentUpdate = true
		oa.Status.Version = oldVersion
		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-disabled"
		pod.Namespace = namespace
		pod.Labels = buildLabels(oaName)
		pod.Spec = newPodSpecForCR(oa, false, consoleLogger, "cluster1")
		pod.Status.HostIP = hostIP
		oa.Status.Tokens = utils.GetTokensName(oa)

		rec := reconciliation{log: consoleLogger, instance: oa, requeueAfter: 30 * time.Minute}
		err := reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		reconciler.reconcileImpl(context.TODO(), &rec)

		assert.NotNil(t, oa.Status.Instances)
		assert.NotEmpty(t, oa.Status.Instances)
	})
}

func NewSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}
