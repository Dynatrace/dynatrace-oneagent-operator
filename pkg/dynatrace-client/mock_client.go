package dynatrace_client

import "github.com/stretchr/testify/mock"

// MockDynatraceClient => mock implementation of Dynatrace Client
type MockDynatraceClient struct {
	mock.Mock
}

// GetAgentVersionForIP => mock GetAgentVersionForIP
func (o *MockDynatraceClient) GetAgentVersionForIP(ip string) (string, error) {
	args := o.Called(ip)
	return args.String(0), args.Error(1)
}

// GetLatestAgentVersion => mock GetLatestAgentVersion
func (o *MockDynatraceClient) GetLatestAgentVersion(os, installerType string) (string, error) {
	args := o.Called(os, installerType)
	return args.String(0), args.Error(1)
}

// GetCommunicationHosts => mock GetCommunicationHosts
func (o *MockDynatraceClient) GetCommunicationHosts() ([]CommunicationHost, error) {
	args := o.Called()
	return args.Get(0).([]CommunicationHost), args.Error(1)
}

// GetCommunicationHostForClient => mock GetCommunicationHostForClient
func (o *MockDynatraceClient) GetCommunicationHostForClient() (CommunicationHost, error) {
	args := o.Called()
	return args.Get(0).(CommunicationHost), args.Error(1)
}

// SendEvent => mock SendEvent
func (o *MockDynatraceClient) SendEvent(event *EventData) (error) {
	args := o.Called(event)
	return args.Error(0)
}

func (o *MockDynatraceClient) GetEntityIDForIP(ip string) (string, error) {
	args := o.Called(ip)
	return args.String(0), args.Error(1)
}
