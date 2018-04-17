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

func TestReadLatesVersion(t *testing.T) {
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
