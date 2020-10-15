package utils

import (
	"fmt"
	"time"

	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/version"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if !instance.GetSpec().UseImmutableImage {
		return false
	}

	status := instance.GetStatus()

	if ts := status.LastClusterVersionProbeTimestamp; ts != nil && !isLastProbeOutdated(ts.UTC()) {
		return false
	}

	now := metav1.Now()
	status.LastClusterVersionProbeTimestamp = &now

	agentVersion, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		logger.Error(err, err.Error())
		return true
	}

	clusterInfo, err := dtc.GetClusterInfo()
	if err != nil {
		logger.Error(err, err.Error())
		return true
	}

	if clusterInfo == nil {
		err = fmt.Errorf("could not retrieve cluster info")
		logger.Error(err, err.Error())
		return true
	}

	status.UseImmutableImage =
		version.IsRemoteClusterVersionSupported(logger, clusterInfo.Version) &&
			version.IsAgentVersionSupported(logger, agentVersion)

	return true
}

func isLastProbeOutdated(ts time.Time) bool {
	return metav1.Now().UTC().Sub(ts) > updateInterval
}
