package dynatrace_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type hostInfo struct {
	version  string
	entityID string
}

// client implements the Client interface.
type dynatraceClient struct {
	url       string
	apiToken  string
	paasToken string

	httpClient *http.Client

	hostCache map[string]hostInfo
}

// makeRequest does an HTTP request by formatting the URL from the given arguments and returns the response.
// The response body must be closed by the caller when no longer used.
func (dc *dynatraceClient) makeRequest(format string, a ...interface{}) (*http.Response, error) {
	url := fmt.Sprintf(format, a...)
	res, err := dc.httpClient.Get(url)
	return res, err
}

func (dc *dynatraceClient) getServerResponseData(response *http.Response) ([]byte, error) {

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error(err, "error reading response")
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		se := &serverError{}
		err := json.Unmarshal(responseData, se)
		if err != nil {
			log.Error(err, "error unmarshalling json response")
			return nil, err
		}
		log.Info("failed to query dynatrace servers", "error", se.Error())

		return nil, errors.New("failed to query dynatrace servers: " + se.Error())
	}

	return responseData, nil
}

func (dc *dynatraceClient) getHostInfoForIP(ip string) (*hostInfo, error) {

	if len(dc.hostCache) == 0 {
		err := dc.buildHostCache()
		if err != nil {
			log.Error(err, "error building hostcache from dynatrace cluster")
			return nil, err
		}
	}

	switch hostInfo, ok := dc.hostCache[ip]; {
	case !ok:
		return nil, errors.New("host not found")
	default:
		return &hostInfo, nil
	}
}

func (dc *dynatraceClient) buildHostCache() error {

	resp, err := dc.makeRequest("%s/v1/entity/infrastructure/hosts?Api-Token=%s&includeDetails=false", dc.url, dc.apiToken)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseData, err := dc.getServerResponseData(resp)
	if err != nil {
		return err
	}

	err = dc.setHostCacheFromResponse(responseData)
	if err != nil {
		return err
	}

	return nil
}

func (dc *dynatraceClient) setHostCacheFromResponse(response []byte) error {
	type hostInfoResponse struct {
		IPAddresses  []string
		AgentVersion *struct {
			Major     int
			Minor     int
			Revision  int
			Timestamp string
		}
		EntityID string
	}

	dc.hostCache = make(map[string]hostInfo)

	hostInfoResponses := []hostInfoResponse{}
	err := json.Unmarshal(response, &hostInfoResponses)
	if err != nil {
		log.Error(err, "error unmarshalling json response")
		return err
	}

	for _, info := range hostInfoResponses {

		hostInfo := hostInfo{entityID: info.EntityID}

		if v := info.AgentVersion; v != nil {
			hostInfo.version = fmt.Sprintf("%d.%d.%d.%s", v.Major, v.Minor, v.Revision, v.Timestamp)
		}
		for _, ip := range info.IPAddresses {
			dc.hostCache[ip] = hostInfo
		}
	}

	return nil
}

// serverError represents an error returned from the server (e.g. authentication failure).
type serverError struct {
	Code    float64
	Message string
}

// Error formats the server error code and message.
func (e *serverError) Error() string {
	if len(e.Message) == 0 && e.Code == 0 {
		return "unknown server error"
	}
	return fmt.Sprintf("error %d: %s", int64(e.Code), e.Message)
}
