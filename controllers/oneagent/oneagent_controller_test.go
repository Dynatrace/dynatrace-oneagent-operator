package oneagent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/dtclient"
	"github.com/Dynatrace/dynatrace-oneagent-operator/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testClusterID = "test-cluster-id"
	testImage     = "test-image"
	testURL       = "https://test-url"
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

func TestUseImmutableImage(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`if image is unset and useImmutableImage is false, default image is used`, func(t *testing.T) {
		instance := dynatracev1alpha1.OneAgent{
			Spec: dynatracev1alpha1.OneAgentSpec{}}
		podSpecs := newPodSpecForCR(&instance, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.Equal(t, defaultOneAgentImage, podSpecs.Containers[0].Image)
	})
	t.Run(`if image is set and useImmutableImage is false, set image is used`, func(t *testing.T) {
		instance := dynatracev1alpha1.OneAgent{
			Spec: dynatracev1alpha1.OneAgentSpec{
				Image: testImage,
			}}
		podSpecs := newPodSpecForCR(&instance, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.Equal(t, testImage, podSpecs.Containers[0].Image)
	})
	t.Run(`if image is set and useImmutableImage is true, set image is used`, func(t *testing.T) {
		instance := dynatracev1alpha1.OneAgent{
			Spec: dynatracev1alpha1.OneAgentSpec{
				BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
					UseImmutableImage: true,
				},
				Image: testImage,
			}}
		podSpecs := newPodSpecForCR(&instance, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.Equal(t, testImage, podSpecs.Containers[0].Image)
	})
	t.Run(`if image is unset and useImmutableImage is true, image is based on api url`, func(t *testing.T) {
		instance := dynatracev1alpha1.OneAgent{
			Spec: dynatracev1alpha1.OneAgentSpec{
				BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
					UseImmutableImage: true,
					APIURL:            testURL,
				},
			},
			Status: dynatracev1alpha1.OneAgentStatus{
				BaseOneAgentStatus: dynatracev1alpha1.BaseOneAgentStatus{
					UseImmutableImage: true,
				},
			}}
		podSpecs := newPodSpecForCR(&instance, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.Equal(t, podSpecs.Containers[0].Image, fmt.Sprintf("%s/linux/oneagent", strings.TrimPrefix(testURL, "https://")))

		instance.Spec.AgentVersion = testValue
		podSpecs = newPodSpecForCR(&instance, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.Equal(t, podSpecs.Containers[0].Image, fmt.Sprintf("%s/linux/oneagent:%s", strings.TrimPrefix(testURL, "https://"), testValue))
	})
}

func TestCustomPullSecret(t *testing.T) {
	log := logger.NewDTLogger()
	instance := dynatracev1alpha1.OneAgent{
		Spec: dynatracev1alpha1.OneAgentSpec{
			BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
				UseImmutableImage: true,
				APIURL:            testURL,
			},
			CustomPullSecret: testName,
		},
		Status: dynatracev1alpha1.OneAgentStatus{
			BaseOneAgentStatus: dynatracev1alpha1.BaseOneAgentStatus{
				UseImmutableImage: true,
			},
		}}
	podSpecs := newPodSpecForCR(&instance, true, log, testClusterID)
	assert.NotNil(t, podSpecs)
	assert.NotEmpty(t, podSpecs.ImagePullSecrets)
	assert.Equal(t, testName, podSpecs.ImagePullSecrets[0].Name)
}

func TestResources(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`minimal cpu request of 100mC is set if no resources specified`, func(t *testing.T) {
		instance := dynatracev1alpha1.OneAgent{
			Spec: dynatracev1alpha1.OneAgentSpec{
				BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
					UseImmutableImage: true,
					APIURL:            testURL,
				},
			},
			Status: dynatracev1alpha1.OneAgentStatus{
				BaseOneAgentStatus: dynatracev1alpha1.BaseOneAgentStatus{
					UseImmutableImage: true,
				},
			}}
		podSpecs := newPodSpecForCR(&instance, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.NotEmpty(t, podSpecs.Containers)

		hasMinimumCPURequest := resource.NewScaledQuantity(1, -1).Equal(*podSpecs.Containers[0].Resources.Requests.Cpu())
		assert.True(t, hasMinimumCPURequest)
	})
	t.Run(`resource requests and limits set`, func(t *testing.T) {
		cpuRequest := resource.NewScaledQuantity(2, -1)
		cpuLimit := resource.NewScaledQuantity(3, -1)
		memoryRequest := resource.NewScaledQuantity(1, 3)
		memoryLimit := resource.NewScaledQuantity(2, 3)

		instance := dynatracev1alpha1.OneAgent{
			Spec: dynatracev1alpha1.OneAgentSpec{
				BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
					UseImmutableImage: true,
					APIURL:            testURL,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    *cpuRequest,
						corev1.ResourceMemory: *memoryRequest,
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    *cpuLimit,
						corev1.ResourceMemory: *memoryLimit,
					},
				},
			},
			Status: dynatracev1alpha1.OneAgentStatus{
				BaseOneAgentStatus: dynatracev1alpha1.BaseOneAgentStatus{
					UseImmutableImage: true,
				},
			}}
		podSpecs := newPodSpecForCR(&instance, true, log, testClusterID)
		assert.NotNil(t, podSpecs)
		assert.NotEmpty(t, podSpecs.Containers)
		hasCPURequest := cpuRequest.Equal(*podSpecs.Containers[0].Resources.Requests.Cpu())
		hasCPULimit := cpuLimit.Equal(*podSpecs.Containers[0].Resources.Limits.Cpu())
		hasMemoryRequest := memoryRequest.Equal(*podSpecs.Containers[0].Resources.Requests.Memory())
		hasMemoryLimit := memoryLimit.Equal(*podSpecs.Containers[0].Resources.Limits.Memory())

		assert.True(t, hasCPURequest)
		assert.True(t, hasCPULimit)
		assert.True(t, hasMemoryRequest)
		assert.True(t, hasMemoryLimit)
	})
}

func TestArguments(t *testing.T) {
	log := logger.NewDTLogger()
	instance := dynatracev1alpha1.OneAgent{
		Spec: dynatracev1alpha1.OneAgentSpec{
			BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
				UseImmutableImage: true,
				APIURL:            testURL,
			},
			Args: []string{testValue},
		},
		Status: dynatracev1alpha1.OneAgentStatus{
			BaseOneAgentStatus: dynatracev1alpha1.BaseOneAgentStatus{
				UseImmutableImage: true,
			},
		}}
	podSpecs := newPodSpecForCR(&instance, true, log, testClusterID)
	assert.NotNil(t, podSpecs)
	assert.NotEmpty(t, podSpecs.Containers)
	assert.Contains(t, podSpecs.Containers[0].Args, testValue)
}

func TestEnvVars(t *testing.T) {
	log := logger.NewDTLogger()
	reservedVariable := "DT_K8S_NODE_NAME"
	instance := dynatracev1alpha1.OneAgent{
		Spec: dynatracev1alpha1.OneAgentSpec{
			BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
				UseImmutableImage: true,
				APIURL:            testURL,
			},
			Env: []corev1.EnvVar{
				{
					Name:  testName,
					Value: testValue,
				},
				{
					Name:  reservedVariable,
					Value: testValue,
				},
			},
		},
		Status: dynatracev1alpha1.OneAgentStatus{
			BaseOneAgentStatus: dynatracev1alpha1.BaseOneAgentStatus{
				UseImmutableImage: true,
			},
		}}
	podSpecs := newPodSpecForCR(&instance, true, log, testClusterID)
	assert.NotNil(t, podSpecs)
	assert.NotEmpty(t, podSpecs.Containers)
	assert.NotEmpty(t, podSpecs.Containers[0].Env)
	assertHasEnvVar(t, testName, testValue, podSpecs.Containers[0].Env)
	assertHasEnvVar(t, reservedVariable, testValue, podSpecs.Containers[0].Env)
}

func assertHasEnvVar(t *testing.T, expectedName string, expectedValue string, envVars []corev1.EnvVar) {
	hasVariable := false
	for _, env := range envVars {
		if env.Name == expectedName {
			hasVariable = true
			assert.Equal(t, expectedValue, env.Value)
		}
	}
	assert.True(t, hasVariable)
}

func TestServiceAccountName(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`has default values`, func(t *testing.T) {
		instance := dynatracev1alpha1.OneAgent{
			Spec: dynatracev1alpha1.OneAgentSpec{
				BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
					UseImmutableImage: true,
					APIURL:            testURL,
				},
			},
			Status: dynatracev1alpha1.OneAgentStatus{
				BaseOneAgentStatus: dynatracev1alpha1.BaseOneAgentStatus{
					UseImmutableImage: true,
				},
			}}
		podSpecs := newPodSpecForCR(&instance, false, log, testClusterID)
		assert.Equal(t, defaultServiceAccountName, podSpecs.ServiceAccountName)

		instance = dynatracev1alpha1.OneAgent{
			Spec: dynatracev1alpha1.OneAgentSpec{
				BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
					UseImmutableImage: true,
					APIURL:            testURL,
				},
			},
			Status: dynatracev1alpha1.OneAgentStatus{
				BaseOneAgentStatus: dynatracev1alpha1.BaseOneAgentStatus{
					UseImmutableImage: true,
				},
			}}
		podSpecs = newPodSpecForCR(&instance, true, log, testClusterID)
		assert.Equal(t, defaultUnprivilegedServiceAccountName, podSpecs.ServiceAccountName)
	})
	t.Run(`uses custom value`, func(t *testing.T) {
		instance := dynatracev1alpha1.OneAgent{
			Spec: dynatracev1alpha1.OneAgentSpec{
				BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
					UseImmutableImage: true,
					APIURL:            testURL,
				},
				ServiceAccountName: testName,
			},
			Status: dynatracev1alpha1.OneAgentStatus{
				BaseOneAgentStatus: dynatracev1alpha1.BaseOneAgentStatus{
					UseImmutableImage: true,
				},
			}}
		podSpecs := newPodSpecForCR(&instance, false, log, testClusterID)
		assert.Equal(t, testName, podSpecs.ServiceAccountName)

		instance = dynatracev1alpha1.OneAgent{
			Spec: dynatracev1alpha1.OneAgentSpec{
				BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
					UseImmutableImage: true,
					APIURL:            testURL,
				},
				ServiceAccountName: testName,
			},
			Status: dynatracev1alpha1.OneAgentStatus{
				BaseOneAgentStatus: dynatracev1alpha1.BaseOneAgentStatus{
					UseImmutableImage: true,
				},
			}}
		podSpecs = newPodSpecForCR(&instance, true, log, testClusterID)
		assert.Equal(t, testName, podSpecs.ServiceAccountName)
	})
}
