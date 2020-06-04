package nodes

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
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const testNamespace = "dynatrace"

var testCacheKey = client.ObjectKey{Name: cacheName, Namespace: testNamespace}

func init() {
	apis.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
	os.Setenv(k8sutil.WatchNamespaceEnvVar, testNamespace)
}

func TestNodesReconciler_CreateCache(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(
		scheme.Scheme,
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}},
		&dynatracev1alpha1.OneAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent1", Namespace: testNamespace},
			Status: dynatracev1alpha1.OneAgentStatus{
				Instances: map[string]dynatracev1alpha1.OneAgentInstance{"node1": {IPAddress: "1.2.3.4"}},
			},
		},
		&dynatracev1alpha1.OneAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent2", Namespace: testNamespace},
			Status: dynatracev1alpha1.OneAgentStatus{
				Instances: map[string]dynatracev1alpha1.OneAgentInstance{"node2": {IPAddress: "5.6.7.8"}},
			},
		})

	dtClient := &dtclient.MockDynatraceClient{}
	defer mock.AssertExpectationsForObjects(t, dtClient)

	ctrl := &ReconcileNodes{
		namespace:    testNamespace,
		client:       fakeClient,
		scheme:       scheme.Scheme,
		logger:       logf.ZapLoggerTo(os.Stdout, true),
		dtClientFunc: utils.StaticDynatraceClient(dtClient),
		local:        true,
	}

	require.NoError(t, ctrl.reconcileAll())

	var cm corev1.ConfigMap
	require.NoError(t, fakeClient.Get(context.TODO(), testCacheKey, &cm))
	nodesCache := &Cache{Obj: &cm}

	if info, err := nodesCache.Get("node1"); assert.NoError(t, err) {
		assert.Equal(t, "1.2.3.4", info.IPAddress)
		assert.Equal(t, "oneagent1", info.Instance)
	}

	if info, err := nodesCache.Get("node2"); assert.NoError(t, err) {
		assert.Equal(t, "5.6.7.8", info.IPAddress)
		assert.Equal(t, "oneagent2", info.Instance)
	}
}

func TestNodesReconciler_DeleteNode(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(
		scheme.Scheme,
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}},
		&dynatracev1alpha1.OneAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent1", Namespace: testNamespace},
			Status: dynatracev1alpha1.OneAgentStatus{
				Instances: map[string]dynatracev1alpha1.OneAgentInstance{"node1": {IPAddress: "1.2.3.4"}},
			},
		},
		&dynatracev1alpha1.OneAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent2", Namespace: testNamespace},
			Status: dynatracev1alpha1.OneAgentStatus{
				Instances: map[string]dynatracev1alpha1.OneAgentInstance{"node2": {IPAddress: "5.6.7.8"}},
			},
		})

	dtClient := &dtclient.MockDynatraceClient{}
	defer mock.AssertExpectationsForObjects(t, dtClient)
	dtClient.On("GetEntityIDForIP", "1.2.3.4", "").Return("HOST-42", nil)
	dtClient.On("SendEvent", mock.MatchedBy(func(e *dtclient.EventData) bool {
		return e.EventType == "MARKED_FOR_TERMINATION"
	})).Return(nil)

	ctrl := &ReconcileNodes{
		namespace:    testNamespace,
		client:       fakeClient,
		scheme:       scheme.Scheme,
		logger:       logf.ZapLoggerTo(os.Stdout, true),
		dtClientFunc: utils.StaticDynatraceClient(dtClient),
		local:        true,
	}

	require.NoError(t, ctrl.reconcileAll())
	require.NoError(t, ctrl.onDeletion("node1"))

	var cm corev1.ConfigMap
	require.NoError(t, fakeClient.Get(context.TODO(), testCacheKey, &cm))
	nodesCache := &Cache{Obj: &cm}

	_, err := nodesCache.Get("node1")
	assert.Equal(t, err, ErrNotFound)

	if info, err := nodesCache.Get("node2"); assert.NoError(t, err) {
		assert.Equal(t, "5.6.7.8", info.IPAddress)
		assert.Equal(t, "oneagent2", info.Instance)
	}
}

func TestNodesReconciler_NodeNotFound(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(
		scheme.Scheme,
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}},
		&dynatracev1alpha1.OneAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent1", Namespace: testNamespace},
			Status: dynatracev1alpha1.OneAgentStatus{
				Instances: map[string]dynatracev1alpha1.OneAgentInstance{"node1": {IPAddress: "1.2.3.4"}},
			},
		},
		&dynatracev1alpha1.OneAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent2", Namespace: testNamespace},
			Status: dynatracev1alpha1.OneAgentStatus{
				Instances: map[string]dynatracev1alpha1.OneAgentInstance{"node2": {IPAddress: "5.6.7.8"}},
			},
		})

	dtClient := &dtclient.MockDynatraceClient{}
	defer mock.AssertExpectationsForObjects(t, dtClient)
	dtClient.On("GetEntityIDForIP", "5.6.7.8", "").Return("HOST-84", nil)
	dtClient.On("SendEvent", mock.MatchedBy(func(e *dtclient.EventData) bool {
		return e.EventType == "MARKED_FOR_TERMINATION"
	})).Return(nil)

	ctrl := &ReconcileNodes{
		namespace:    testNamespace,
		client:       fakeClient,
		scheme:       scheme.Scheme,
		logger:       logf.ZapLoggerTo(os.Stdout, true),
		dtClientFunc: utils.StaticDynatraceClient(dtClient),
		local:        true,
	}

	require.NoError(t, ctrl.reconcileAll())
	var node2 corev1.Node
	require.NoError(t, fakeClient.Get(context.TODO(), client.ObjectKey{Name: "node2"}, &node2))
	require.NoError(t, fakeClient.Delete(context.TODO(), &node2))
	require.NoError(t, ctrl.reconcileAll())

	var cm corev1.ConfigMap
	require.NoError(t, fakeClient.Get(context.TODO(), testCacheKey, &cm))
	nodesCache := &Cache{Obj: &cm}

	if info, err := nodesCache.Get("node1"); assert.NoError(t, err) {
		assert.Equal(t, "1.2.3.4", info.IPAddress)
		assert.Equal(t, "oneagent1", info.Instance)
	}

	_, err := nodesCache.Get("node2")
	assert.Equal(t, err, ErrNotFound)
}
