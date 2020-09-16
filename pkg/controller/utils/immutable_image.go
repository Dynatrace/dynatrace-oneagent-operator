package utils

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/version"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const updateInterval = 5 * time.Minute

// SetUseImmutableImageStatus sets the UseImmutableImage and LastClusterVersionProbeTimestamp stati of an BaseOneAgentDaemonSet instance
// Returns true if:
//     UseImmutableImage of specification is true &&
//			LastClusterVersionProbeTimestamp status is the duration of updateInterval behind
// otherwise returns false
func SetUseImmutableImageStatus(logger logr.Logger, instance v1alpha1.BaseOneAgent, dtc dtclient.Client) bool {
	if dtc == nil {
		err := fmt.Errorf("dynatrace client is nil")
		logger.Error(err, err.Error())
		return false
	}

	// Variable declared to make if-condition more readable
	lastClusterVersionProbeTimestamp := instance.GetStatus().LastClusterVersionProbeTimestamp.UTC()
	if instance.GetSpec().UseImmutableImage && isLastProbeOutdated(lastClusterVersionProbeTimestamp) {
		instance.GetStatus().LastClusterVersionProbeTimestamp = metav1.Now()
		agentVersion, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
		if err == nil {
			instance.GetStatus().UseImmutableImage =
				version.IsRemoteClusterVersionSupported(logger, dtc) &&
					version.IsAgentVersionSupported(logger, agentVersion)
		} else {
			logger.Error(err, err.Error())
		}
		return true
	}
	return false
}

func isLastProbeOutdated(lastClusterVersionProbeTimestamp time.Time) bool {
	return metav1.Now().UTC().Sub(lastClusterVersionProbeTimestamp) > updateInterval
}
