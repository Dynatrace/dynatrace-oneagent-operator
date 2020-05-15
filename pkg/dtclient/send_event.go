package dtclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	MarkedForTerminationEvent = "MARKED_FOR_TERMINATION"
)

// EventData struct which defines what event payload should contain
type EventData struct {
	EventType     string               `json:"eventType"`
	StartInMillis uint64               `json:"start"`
	EndInMillis   uint64               `json:"end"`
	Description   string               `json:"description"`
	AttachRules   EventDataAttachRules `json:"attachRules"`
	Source        string               `json:"source"`
}

type EventDataAttachRules struct {
	EntityIDs []string `json:"entityIds"`
}

// EventResponse is the response when sending events to the Dynatrace API.
type EventResponse struct {
	StoredEventIds       []int64  `json:"storedEventIds"`
	StoredIds            []string `json:"storedIds"`
	StoredCorrelationIDs []string `json:"storedCorrelationIds"`
}

func (dc *dynatraceClient) SendEvent(eventData *EventData) (*EventResponse, error) {
	if eventData == nil {
		return nil, errors.New("no data found in eventData payload")
	}

	if eventData.EventType == "" {
		return nil, errors.New("no key set for eventType in eventData payload")
	}

	jsonStr, err := json.Marshal(eventData)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/events", dc.url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, fmt.Errorf("error initialising http request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token %s", dc.apiToken))

	response, err := dc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	data, err := dc.getServerResponseData(response)
	if err != nil {
		return nil, fmt.Errorf("error gathering dynatrace api response: %w", err)
	}

	var er EventResponse
	if err := json.Unmarshal(data, &er); err != nil {
		return nil, fmt.Errorf("fail to parse dynatrace api response: %w", err)
	}

	return &er, nil
}
