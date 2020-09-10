package utils

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	"regexp"
	"strconv"
)

var minSupportedClusterVersion = &clusterVersionInfo{
	major:   1,
	minor:   198,
	release: 0,
}

type clusterVersionInfo struct {
	major   int
	minor   int
	release int
}

func IsRemoteSupportedClusterVersion(logger logr.Logger, dtc dtclient.Client) bool {
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

	isSupported, err := isSupportedClusterVersion(remoteVersion)
	if err != nil {
		logger.Error(err, err.Error())
		return false
	}

	return isSupported
}

func isSupportedClusterVersion(clusterVersion *clusterVersionInfo) (bool, error) {
	if clusterVersion == nil {
		return false, nil
	}

	comparison, err := compareClusterVersion(clusterVersion, minSupportedClusterVersion)
	return comparison >= 0, err
}

// compareClusterVersion returns:
// 	0: if a == b
//  n > 0: if a > b
//  n < 0: if a < b
//  0 with error: if a == nil || b == nil
func compareClusterVersion(a *clusterVersionInfo, b *clusterVersionInfo) (int, error) {
	if a == nil || b == nil {
		return 0, fmt.Errorf("parameter must not be null")
	}

	// Check major version
	result := a.major - b.major
	if result != 0 {
		return result, nil
	}

	// Major is equal, check minor
	result = a.minor - b.minor
	if result != 0 {
		return result, nil
	}

	// Major and minor is equal, check release
	result = a.release - b.release
	return result, nil
}

func extractVersion(versionString string) (*clusterVersionInfo, error) {
	mainVersionRegex := regexp.MustCompile(`[\d]+\.[\d]+\.[\d]+`)
	mainVersion := mainVersionRegex.FindString(versionString)

	versionMergeRegex := regexp.MustCompile(`[\d]+`)
	versions := versionMergeRegex.FindAllString(mainVersion, 3)
	if len(versions) < 3 {
		return nil, fmt.Errorf("version malformed: %s", versionString)
	}

	major, err := strconv.Atoi(versions[0])
	if err != nil {
		return nil, err
	}

	minor, err := strconv.Atoi(versions[1])
	if err != nil {
		return nil, err
	}

	release, err := strconv.Atoi(versions[2])
	if err != nil {
		return nil, err
	}

	return &clusterVersionInfo{major, minor, release}, nil
}
