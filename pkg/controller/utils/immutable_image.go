package utils

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const updateInterval = 5 * time.Minute

type Operator interface {
	Reconcile(request reconcile.Request) (reconcile.Result, error)
}

// SetUseImmutableImageStatus sets the UseImmutableImage and LastClusterVersionProbeTimestamp stati of an BaseOneAgentDaemonSet instance
// Returns true if:
//     UseImmutableImage of specification is true &&
//			LastClusterVersionProbeTimestamp status is the duration of updateInterval behind
// otherwise returns false
func SetUseImmutableImageStatus(instance v1alpha1.BaseOneAgent, logger logr.Logger, dtc dtclient.Client) bool {
	if dtc == nil {
		err := fmt.Errorf("dynatrace client is nil")
		logger.Error(err, err.Error())
		return false
	}
	if instance.GetSpec().UseImmutableImage &&
		metav1.Now().UTC().Sub(instance.GetStatus().LastClusterVersionProbeTimestamp.UTC()) > updateInterval {
		instance.GetStatus().LastClusterVersionProbeTimestamp = metav1.Now()
		instance.GetStatus().UseImmutableImage =
			instance.GetSpec().UseImmutableImage && IsRemoteSupportedClusterVersion(logger, dtc)
		return true
	}
	return false
}
