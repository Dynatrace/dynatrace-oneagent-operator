package dynatrace_client

import (
	"encoding/json"
	"net/http"
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

func testSendEvent(t *testing.T, dynatraceClient Client) {
	{
		testValidEventData := []byte(`{
			"eventType": "CUSTOM_ANNOTATION",
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
		err := json.Unmarshal(testValidEventData, &testEventData)
		assert.NoError(t, err)

		err = dynatraceClient.SendEvent(&testEventData)
		assert.NoError(t, err)
	}
	{
		testInvalidEventData := []byte(`{
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
		err := json.Unmarshal(testInvalidEventData, &testEventData)
		assert.NoError(t, err)

		err = dynatraceClient.SendEvent(&testEventData)
		assert.Error(t, err, "no eventType set")
	}
	{
		testExtraKeysEventData := []byte(`{
			"eventType": "CUSTOM_EVENT_TYPE",
			"extraKey" : "extraKey", 
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
		err := json.Unmarshal(testExtraKeysEventData, &testEventData)
		assert.NoError(t, err)

		err = dynatraceClient.SendEvent(&testEventData)
		assert.NoError(t, err)
	}
}

func handleSendEvent(request *http.Request, writer http.ResponseWriter) {
	eventPostResponse := []byte(`{
		"storedEventIds": [1],
		"storedIds": ["string"],
		"storedCorrelationIds": ["string"]}`)

	switch request.Method {
	case "POST":
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(eventPostResponse))
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}
