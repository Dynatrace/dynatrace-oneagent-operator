package oneagenttests

// declaring constants here because golang doesn't find them if
// test classes are executed separately
const (
	testImage        = "test-image:latest"
	namespace        = "dynatrace"
	testName         = "test-name"
	keyApiURL        = "DYNATRACE_API_URL"
	maxWaitCycles    = 5
	keyEnvironmentId = "DYNATRACE_ENVIRONMENT_ID"
	keySkipCertCheck = "ONEAGENT_INSTALLER_SKIP_CERT_CHECK"
)
