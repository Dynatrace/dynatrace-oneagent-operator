package dynatrace_client

import "github.com/stretchr/testify/mock"

// MockDynatraceClient => mock implementation of Dynatrace Client
type MockDynatraceClient struct {
	mock.Mock
}

// GetVersionForIp => mock GetVersionForIp
func (o *MockDynatraceClient) GetVersionForIp(ip string) (string, error) {
	args := o.Called(ip)
	return args.String(0), args.Error(1)
}

// GetVersionForLatest => mock GetVersionForLatest
func (o *MockDynatraceClient) GetVersionForLatest(os, installerType string) (string, error) {
	args := o.Called(os, installerType)
	return args.String(0), args.Error(1)
}

// GetCommunicationHosts => mock GetCommunicationHosts
func (o *MockDynatraceClient) GetCommunicationHosts() ([]CommunicationHost, error) {
	args := o.Called()
	return args.Get(0).([]CommunicationHost), args.Error(1)
}

// GetAPIURLHost => mock GetAPIURLHost
func (o *MockDynatraceClient) GetAPIURLHost() (CommunicationHost, error) {
	args := o.Called()
	return args.Get(0).(CommunicationHost), args.Error(1)
}

// PostMarkedForTermination => mock GetVersionForIp
func (o *MockDynatraceClient) PostMarkedForTermination(nodeID string) (string, error) {
	args := o.Called(nodeID)
	return args.String(0), args.Error(1)
}
