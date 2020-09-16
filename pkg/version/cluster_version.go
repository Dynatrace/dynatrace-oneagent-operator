package version

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/go-logr/logr"
)

var minSupportedClusterVersion = versionInfo{
	major:   1,
	minor:   198,
	release: 0,
}

func IsRemoteClusterVersionSupported(logger logr.Logger, dtc dtclient.Client) bool {
	if dtc == nil {
		err := fmt.Errorf("dtclient is null")
		logger.Error(err, err.Error())
		return false
	}

	clusterInfo, err := dtc.GetClusterInfo()
	if err != nil {
		logger.Error(err, err.Error())
		return false
	}

	remoteVersion, err := extractVersion(clusterInfo.Version)
	if err != nil {
		logger.Error(err, err.Error())
		return false
	}

	return isSupportedClusterVersion(remoteVersion)
}

func isSupportedClusterVersion(clusterVersion versionInfo) bool {
	comparison := compareVersionInfo(clusterVersion, minSupportedClusterVersion)
	return comparison >= 0
}
