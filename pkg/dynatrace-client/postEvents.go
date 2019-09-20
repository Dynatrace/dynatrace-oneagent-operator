package dynatrace_client

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
)

// PostMarkedForTerminationEvent =>
// send event to dynatrace api that an event has been marked for termination
func (dc *dynatraceClient) PostMarkedForTerminationEvent(nodeIP string) (string, error) {

	hostInfo, err := dc.getHostInfoForIP(nodeIP)
	if err != nil {
		return "", err
	}
	if hostInfo.entityID == "" {
		return "", errors.New("entity ID not set for host")
	}

	url := fmt.Sprintf("%s/v1/events", dc.url)

	body := `
	{
		"eventType": "MARKED_FOR_TERMINATION",
		"timeoutMinutes": 20,
		"attachRules": {
		  "entityIds": [
			%s
		  ]
		},
		"source": "OneAgent Operator",
		"annotationDescription": "K8s node was marked unschedulable. Node is likely being drained"
	  }
	`
	bbytes := []byte(fmt.Sprintf(body, hostInfo.entityID))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bbytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := dc.httpClient.Do(req)
	if err != nil {
		log.Error(err, "error making POST request to dynatrace server")
		return "", errors.New("could not send event MARKED_FOR_TERMINATION to dynatrace server")
	}
	defer resp.Body.Close()
	return resp.Status, nil
}
