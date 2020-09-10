package version

import (
	"fmt"
	"regexp"
	"strconv"
)

type versionInfo struct {
	major   int
	minor   int
	release int
}

// compareVersionInfo returns:
// 	0: if a == b
//  n > 0: if a > b
//  n < 0: if a < b
//  0 with error: if a == nil || b == nil
func compareVersionInfo(a *versionInfo, b *versionInfo) (int, error) {
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

func extractVersion(versionString string) (*versionInfo, error) {
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

	return &versionInfo{major, minor, release}, nil
}
