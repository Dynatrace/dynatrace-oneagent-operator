package dynatrace_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	MarkForTerminationEvent = "MARK_FOR_TERMINATION"
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

	AttachRules EventDataAttachRules `json:"attachRules"`
}

type EventDataAttachRules struct {
	EntityIDs []string `json:"entityIds"`
}

func (dc *dynatraceClient) SendEvent(eventData *EventData) error {
	if eventData == nil {
		err := errors.New("no data found in eventData payload")
		logger.Error(err, "error reading payload")
		return err
	}

	if eventData.EventType == "" {
		err := errors.New("no key set for eventType in eventData payload")
		logger.Error(err, "error reading payload")
		return err
	}

	jsonStr, err := json.Marshal(eventData)
	if err != nil {
		logger.Error(err, "error marshalling eventData payload to json")
		return err
	}

	url := fmt.Sprintf("%s/v1/events", dc.url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Error(err, "error initialising http request")
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token: %s", dc.apiToken))

	resp, err := dc.httpClient.Do(req)
	if err != nil {
		logger.Error(err, "error making post request tp dynatrace api")
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unwanted status code returned %v", resp.StatusCode)
		logger.Error(err, "error received from dynatrace api")
		return err
	}

	return nil
}
