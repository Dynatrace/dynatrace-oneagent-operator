package dynatrace_client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventDataMarshal(t *testing.T) {
	testJSONInput := []byte(`{
	"eventType": "CUSTOM_ANNOTATION",
	"start": 1521042929000,
	"end": 1521542929000,
	"timeoutMinutes": 2,
	"attachRules": {
	  "entityIds": [
		"CUSTOM_DEVICE-0000000000000007"
	  ]
	},
	"source": "OpsControl",
	"annotationType": "defect",
	"annotationDescription": "The coffee machine is broken"
  }`)

	var testEventData EventData
	err := json.Unmarshal(testJSONInput, &testEventData)
	assert.NoError(t, err)
	assert.Equal(t, testEventData.EventType, "CUSTOM_ANNOTATION")
	assert.ElementsMatch(t, testEventData.AttachRules.EntityIDs, []string{"CUSTOM_DEVICE-0000000000000007"})
	assert.Equal(t, testEventData.Source, "OpsControl")

	jsonBuffer, err := json.Marshal(testEventData)
	assert.NoError(t, err)
	assert.JSONEq(t, string(jsonBuffer), string(testJSONInput))
}
