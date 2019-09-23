package dynatrace_client

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testCommunicationHostsGetCommunicationHosts(t *testing.T, dynatraceClient Client) {

	res, err := dynatraceClient.GetCommunicationHosts()

	assert.NoError(t, err)
	assert.ObjectsAreEqualValues(res, []CommunicationHost{
		CommunicationHost{Host: "host1.dynatracelabs.com", Port: 80, Protocol: "http"},
		CommunicationHost{Host: "host2.dynatracelabs.com", Port: 443, Protocol: "https"},
		CommunicationHost{Host: "12.0.9.1", Port: 80, Protocol: "http"},
	})
}

func handleCommunicationHosts(request *http.Request, writer http.ResponseWriter) {
	commHostOutput := []byte(`{
		"tenantUUID": "string",
		"tenantToken": "string",
		"communicationEndpoints": [
		  "http://host1.domain.com",
		  "https://host2.domain.com",
		  "http://host3.domain.com",
		  "http://12.0.9.1",
		  "http://12.0.10.1"
		]
	}`)

	switch request.Method {
	case "GET":
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(commHostOutput))
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}
