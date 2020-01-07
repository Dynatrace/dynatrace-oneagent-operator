package dtclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeRequest(t *testing.T) {
	dynatraceServer := httptest.NewServer(dynatraceServerHandler())
	defer dynatraceServer.Close()

	dc := &dynatraceClient{
		url:       dynatraceServer.URL,
		apiToken:  apiToken,
		paasToken: paasToken,

		hostCache:  make(map[string]hostInfo),
		httpClient: http.DefaultClient,
	}

	require.NotNil(t, dc)

	{
		var url string = fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo", dc.url)
		resp, err := dc.makeRequest(url, dynatraceApiToken)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	}
	{
		resp, err := dc.makeRequest("%s/v1/deployment/installer/agent/connectioninfo", dynatraceApiToken)
		assert.Error(t, err, "unsupported protocol scheme")
		assert.Nil(t, resp)
	}
}

func TestGetResponseOrServerError(t *testing.T) {
	dynatraceServer := httptest.NewServer(dynatraceServerHandler())
	defer dynatraceServer.Close()

	dc := &dynatraceClient{
		url:       dynatraceServer.URL,
		apiToken:  apiToken,
		paasToken: paasToken,

		hostCache:  make(map[string]hostInfo),
		httpClient: http.DefaultClient,
	}

	require.NotNil(t, dc)

	reqURL := fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo", dc.url)
	{
		resp, err := dc.makeRequest(reqURL, dynatraceApiToken)
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		body, serverError, err := dc.getServerResponseData(resp)
		assert.NoError(t, err)
		assert.Nil(t, serverError)
		assert.NotNil(t, body, "response body available")
	}
}

func TestBuildHostCache(t *testing.T) {
	dynatraceServer := httptest.NewServer(dynatraceServerHandler())
	defer dynatraceServer.Close()

	dc := &dynatraceClient{
		url:       dynatraceServer.URL,
		paasToken: paasToken,

		hostCache:  make(map[string]hostInfo),
		httpClient: http.DefaultClient,
	}

	require.NotNil(t, dc)

	{
		err := dc.buildHostCache()
		assert.Error(t, err, "error querying dynatrace server")
		assert.Empty(t, dc.hostCache)
	}
	{
		dc.apiToken = apiToken
		err := dc.buildHostCache()
		assert.NoError(t, err)
		assert.NotZero(t, len(dc.hostCache))
		assert.ObjectsAreEqualValues(dc.hostCache, map[string]hostInfo{
			"10.11.12.13": hostInfo{version: "1.142.0.20180313-173634", entityID: "dynatraceSampleEntityId"},
			"192.168.0.1": hostInfo{version: "1.142.0.20180313-173634", entityID: "dynatraceSampleEntityId"},
		})
	}
}

func TestServerError(t *testing.T) {
	{
		se := &serverError{
			ErrorMessage: struct {
				Code    int64
				Message string
			}{
				Code:    401,
				Message: "Unauthorized",
			},
		}
		assert.Equal(t, se.Error(), "error 401: Unauthorized")
	}
	{
		se := &serverError{
			ErrorMessage: struct {
				Code    int64
				Message string
			}{
				Message: "Unauthorized",
			},
		}
		assert.Equal(t, se.Error(), "error 0: Unauthorized")
	}
	{
		se := &serverError{
			ErrorMessage: struct {
				Code    int64
				Message string
			}{
				Code: 401,
			},
		}
		assert.Equal(t, se.Error(), "error 401: ")
	}
	{
		se := &serverError{}
		assert.Equal(t, se.Error(), "unknown server error")
	}
}

func TestDynatraceClientWithServer(t *testing.T) {
	dynatraceServer := httptest.NewServer(dynatraceServerHandler())
	defer dynatraceServer.Close()

	skipCert := SkipCertificateValidation(true)
	dynatraceClient, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)

	require.NoError(t, err)
	require.NotNil(t, dynatraceClient)

	testAgentVersionGetLatestAgentVersion(t, dynatraceClient)
	testAgentVersionGetAgentVersionForIP(t, dynatraceClient)
	testCommunicationHostsGetCommunicationHosts(t, dynatraceClient)
	testSendEvent(t, dynatraceClient)
}

func dynatraceServerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.FormValue("Api-Token") == "" && r.Header.Get("Authorization") == "" {
			writeError(w, http.StatusUnauthorized)
		} else {
			handleRequest(r, w)
		}
	}
}

func handleRequest(request *http.Request, writer http.ResponseWriter) {
	latestAgentVersion := fmt.Sprintf("/v1/deployment/installer/agent/%s/%s/latest/metainfo", OsUnix, InstallerTypeDefault)
	versionForIP := fmt.Sprint("/v1/entity/infrastructure/hosts")
	communicationHosts := fmt.Sprint("/v1/deployment/installer/agent/connectioninfo")
	sendEvent := fmt.Sprint("/v1/events")

	switch request.URL.Path {
	case latestAgentVersion:
		handleLatestAgentVersion(request, writer)
	case versionForIP:
		handleVersionForIP(request, writer)
	case communicationHosts:
		handleCommunicationHosts(request, writer)
	case sendEvent:
		handleSendEvent(request, writer)
	default:
		writeError(writer, http.StatusBadRequest)
	}
}

func writeError(w http.ResponseWriter, status int) {
	message := serverError{
		ErrorMessage: struct {
			Code    int64
			Message string
		}{
			Code:    int64(status),
			Message: "error received from server",
		},
	}
	result, _ := json.Marshal(&message)

	w.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = w.Write(result)
}
