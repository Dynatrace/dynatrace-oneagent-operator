package dynatrace_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// EventData struct which defines what event payload should contain
type EventData struct {
	EventType             string `json:"eventType"`
	Source                string `json:"source"`
	AnnotationType        string `json:"annotationType"`
	AnnotationDescription string `json:"annotationDescription"`

	End            float64 `json:"end"`
	Start          float64 `json:"start"`
	TimeoutMinutes float64 `json:"timeoutMinutes"`

	AttachRules struct {
		EntityIDs []string `json:"entityIds"`
	} `json:"attachRules"`
}

func (dc *dynatraceClient) SendEvent(eventData *EventData) error {

	if eventData == nil {
		err := errors.New("no data found in eventData payload")
		log.Error(err, "error reading payload")
		return err
	}

	if eventData.EventType == "" {
		err := errors.New("no key set for eventType in eventData payload")
		log.Error(err, "error reading payload")
		return err
	}

	jsonStr, err := json.Marshal(eventData)
	if err != nil {
		log.Error(err, "error marshalling eventData payload to json")
		return err
	}

	url := fmt.Sprintf("%s/v1/events", dc.url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Error(err, "error initialising http request")
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token: %s", dc.apiToken))

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
