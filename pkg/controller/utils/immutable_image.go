package utils

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/version"
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
func SetUseImmutableImageStatus(logger logr.Logger, instance v1alpha1.BaseOneAgent, agentVersion string, dtc dtclient.Client) bool {
	if dtc == nil {
		err := fmt.Errorf("dynatrace client is nil")
		logger.Error(err, err.Error())
		return false
	}

	if instance.GetSpec().UseImmutableImage &&
		metav1.Now().UTC().Sub(instance.GetStatus().LastClusterVersionProbeTimestamp.UTC()) > updateInterval {
		instance.GetStatus().LastClusterVersionProbeTimestamp = metav1.Now()
		instance.GetStatus().UseImmutableImage =
			instance.GetSpec().UseImmutableImage &&
				version.IsRemoteClusterVersionSupported(logger, dtc) &&
				version.IsAgentVersionSupported(logger, agentVersion)
		return true
	}
	return false
}
