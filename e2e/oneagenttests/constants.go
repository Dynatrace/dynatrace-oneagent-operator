package oneagenttests

// declaring constants here because golang doesn't find them if
// test classes are executed separately
const (
	testImage    = "test-image:latest"
	testName     = "test-name"
	testData     = "test-data"
	testCertName = "certs"

	namespace           = "dynatrace"
	maxWaitCycles       = 5
	trustedCertPath     = "/mnt/dynatrace/certs"
	trustedCertFilename = "certs.pem"

	keyApiURL        = "DYNATRACE_API_URL"
	keySkipCertCheck = "ONEAGENT_INSTALLER_SKIP_CERT_CHECK"
)
