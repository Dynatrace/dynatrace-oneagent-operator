package dynatrace_client

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar")
		if assert.NoError(t, err) {
			assert.NotNil(t, c)
		}
	}
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar", SkipCertificateValidation(false))
		if assert.NoError(t, err) {
			assert.NotNil(t, c)
		}
	}
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar", SkipCertificateValidation(true))
		if assert.NoError(t, err) {
			assert.NotNil(t, c)
		}
	}

	{
		_, err := NewClient("https://aabb.live.dynatrace.com/api", "", "foo")
		assert.Error(t, err, "empty API token")
	}
	{
		_, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "")
		assert.Error(t, err, "empty PaaS token")
	}
	{
		_, err := NewClient("", "foo", "bar")
		assert.Error(t, err, "empty URL")
	}
}

func TestClient_GetVersionForLatest(t *testing.T) {
	c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar")
	require.NoError(t, err)
	require.NotNil(t, c)

	{
		_, err = c.GetVersionForLatest("", "default")
		assert.Error(t, err, "empty OS")
	}
	{
		_, err = c.GetVersionForLatest("unix", "")
		assert.Error(t, err, "empty installer type")
	}
}

func TestClient_GetVersionForIp(t *testing.T) {
	c := func() Client {
		c := client{
			url:       "https://aabb.live.dynatrace.com/api",
			apiToken:  "foo",
			paasToken: "bar",
		}
		hosts, err := readHostMap(strings.NewReader(goodHostsResponse))
		require.NoError(t, err)
		c.hostCache = hosts
		return &c
	}()

	{
		v, err := c.GetVersionForIp(goodIp)
		if assert.NoError(t, err) {
			assert.Equal(t, "1.142.0.20180313-173634", v)
		}
	}

	{
		_, err := c.GetVersionForIp("")
		assert.Error(t, err, "empty IP")
	}

	{
		_, err := c.GetVersionForIp(unknownIp)
		assert.Error(t, err, "unknown host")
	}
	{
		_, err := c.GetVersionForIp(unsetIp)
		assert.Error(t, err, "no version")
	}
}

func TestReadLatestVersion(t *testing.T) {
	readFromString := func(json string) (string, error) {
		r := strings.NewReader(json)
		return readLatestVersion(r)
	}

	{
		v, err := readFromString(`{"latestAgentVersion":"1.122.0.20170101-101010"}`)
		if assert.NoError(t, err) {
			assert.Equal(t, "1.122.0.20170101-101010", v)
		}
	}

	{
		_, err := readFromString("")
		assert.Error(t, err, "empty response")
	}
	{
		_, err := readFromString(`{"latestAgentVersion":null}`)
		assert.Error(t, err, "null version")
	}
	{
		_, err := readFromString(`{"latestAgentVersion":""}`)
		assert.Error(t, err, "empty version")
	}
	{
		_, err := readFromString(`{"error":{"code":401,"message":"Token Authentication failed"}}`)
		if assert.Error(t, err, "server error") {
			assert.Contains(t, err.Error(), "401")
			assert.Contains(t, err.Error(), "Token Authentication failed")
		}
	}
}

const goodHostsResponse = `[
  {
    "displayName": "good",
    "ipAddresses": [
      "10.11.12.13",
      "192.168.0.1"
    ],
    "agentVersion": {
      "major": 1,
      "minor": 142,
      "revision": 0,
      "timestamp": "20180313-173634"
    }
  },
  {
    "displayName": "unset version",
    "ipAddresses": [
      "192.168.100.1"
    ]
  }
]`

const (
	goodIp    = "192.168.0.1"
	unsetIp   = "192.168.100.1"
	unknownIp = "127.0.0.1"
)

func TestReadHostMap(t *testing.T) {
	readFromString := func(json string) (map[string]string, error) {
		r := strings.NewReader(json)
		return readHostMap(r)
	}

	{
		m, err := readFromString(goodHostsResponse)
		if assert.NoError(t, err) {
			expected := map[string]string{
				"10.11.12.13":   "1.142.0.20180313-173634",
				"192.168.0.1":   "1.142.0.20180313-173634",
				"192.168.100.1": "",
			}
			assert.Equal(t, expected, m)
		}
	}

	{
		_, err := readFromString("")
		assert.Error(t, err, "empty response")
	}
	{
		m, err := readFromString("[]")
		if assert.NoError(t, err, "no hosts") {
			assert.Equal(t, 0, len(m))
		}
	}
	{
		_, err := readFromString(`{"error":{"code":401,"message":"Token Authentication failed"}}`)
		if assert.Error(t, err, "server error") {
			assert.Contains(t, err.Error(), "401")
			assert.Contains(t, err.Error(), "Token Authentication failed")
		}
	}
}

const goodCommunicationEndpointsResponse = `{
	"tenantUUID": "aabb",
	"tenantToken": "testtoken",
	"communicationEndpoints": [
		"https://example.live.dynatrace.com/communication",
		"https://managedhost.com:9999/here/communication",
		"https://10.0.0.1:8000/communication",
		"http://insecurehost/communication"
	]
}`

const mixedCommunicationEndpointsResponse = `{
	"tenantUUID": "aabb",
	"tenantToken": "testtoken",
	"communicationEndpoints": [
		"https://example.live.dynatrace.com/communication",
		"https://managedhost.com:notaport/here/communication",
		"example.live.dynatrace.com:80/communication",
		"ftp://randomhost.com:80/communication",
		"unix:///some/local/file",
		"shouldnotbeparsed"
	]
}`

func TestReadCommunicationHosts(t *testing.T) {
	readFromString := func(json string) ([]CommunicationHost, error) {
		r := strings.NewReader(json)
		return readCommunicationHosts(r)
	}

	{
		m, err := readFromString(goodCommunicationEndpointsResponse)
		if assert.NoError(t, err) {
			expected := []CommunicationHost{
				{Protocol: "https", Host: "example.live.dynatrace.com", Port: 443},
				{Protocol: "https", Host: "managedhost.com", Port: 9999},
				{Protocol: "https", Host: "10.0.0.1", Port: 8000},
				{Protocol: "http", Host: "insecurehost", Port: 80},
			}
			assert.Equal(t, expected, m)
		}
	}

	{
		m, err := readFromString(mixedCommunicationEndpointsResponse)
		if assert.NoError(t, err) {
			expected := []CommunicationHost{
				{Protocol: "https", Host: "example.live.dynatrace.com", Port: 443},
			}
			assert.Equal(t, expected, m)
		}
	}

	{
		_, err := readFromString("")
		assert.Error(t, err, "empty response")
	}
	{
		_, err := readFromString(`{"error":{"code":401,"message":"Token Authentication failed"}}`)
		if assert.Error(t, err, "server error") {
			assert.Contains(t, err.Error(), "401")
			assert.Contains(t, err.Error(), "Token Authentication failed")
		}
	}
	{
		_, err := readFromString(`{"communicationEndpoints": ["shouldnotbeparsed"]}`)
		assert.Error(t, err, "no hosts available")
	}
}

func TestCommunicationHostParsing(t *testing.T) {
	var err error
	var ch CommunicationHost

	// Successful parsing

	ch, err = parseEndpoint("https://example.live.dynatrace.com/communication")
	assert.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "example.live.dynatrace.com",
		Port:     443,
	}, ch)

	ch, err = parseEndpoint("https://managedhost.com:9999/here/communication")
	assert.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "managedhost.com",
		Port:     9999,
	}, ch)

	ch, err = parseEndpoint("https://example.live.dynatrace.com/communication")
	assert.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "example.live.dynatrace.com",
		Port:     443,
	}, ch)

	ch, err = parseEndpoint("https://10.0.0.1:8000/communication")
	assert.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "https",
		Host:     "10.0.0.1",
		Port:     8000,
	}, ch)

	ch, err = parseEndpoint("http://insecurehost/communication")
	assert.NoError(t, err)
	assert.Equal(t, CommunicationHost{
		Protocol: "http",
		Host:     "insecurehost",
		Port:     80,
	}, ch)

	// Failures

	_, err = parseEndpoint("https://managedhost.com:notaport/here/communication")
	assert.Error(t, err)

	_, err = parseEndpoint("example.live.dynatrace.com:80/communication")
	assert.Error(t, err)

	_, err = parseEndpoint("ftp://randomhost.com:80/communication")
	assert.Error(t, err)

	_, err = parseEndpoint("unix:///some/local/file")
	assert.Error(t, err)

	_, err = parseEndpoint("shouldnotbeparsed")
	assert.Error(t, err)
}
