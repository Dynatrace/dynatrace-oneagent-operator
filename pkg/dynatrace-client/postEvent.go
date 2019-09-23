package dynatrace_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

func (dc *dynatraceClient) SendEvent(payload map[string]interface{}) error {

	if len(payload) == 0 {
		err := errors.New("no data found in payload")
		log.Error(err, "error reading payload")
		return err
	}

	if _, ok := payload["eventType"]; ok == false {
		err := errors.New("no key set for eventType in payload")
		log.Error(err, "error reading payload")
		return err
	}

	jsonStr, err := json.Marshal(payload)
	if err != nil {
		log.Error(err, "error marshalling payload to json")
		return err
	}

	url := fmt.Sprintf("%s/v1/events", dc.url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Error(err, "error initialising http request")
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Authorization: %s", dc.apiToken))

	resp, err := dc.httpClient.Do(req)
	if err != nil {
		log.Error(err, "error making post request tp dynatrace api")
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unwanted status code returned %v", resp.StatusCode)
		log.Error(err, "error received from dynatrace api")
		return err
	}

	return nil
}
