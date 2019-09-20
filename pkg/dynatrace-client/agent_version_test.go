package dynatrace_client

import (
	"testing"
)

func testDynatraceClientGetLatestAgentVersion(t *testing.T, dtc Client) {

	something, err := dtc.GetLatestAgentVersion(OsUnix, InstallerTypeDefault)
	t.Error(something)
	t.Error(err)
	// t.Error(res, err)
}
