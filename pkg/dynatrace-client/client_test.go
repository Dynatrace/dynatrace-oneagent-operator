package dynatrace_client

import (
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	assert.NotPanics(t, func() {
		c := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar")
		assert.NotNil(t, c)
	})

	assert.Panics(t, func() {
		NewClient("https://aabb.live.dynatrace.com/api", "", "foo")
	}, "empty API token")

	assert.Panics(t, func() {
		NewClient("https://aabb.live.dynatrace.com/api", "foo", "")
	}, "empty PaaS token")

	assert.Panics(t, func() {
		NewClient("", "foo", "bar")
	}, "empty URL")
}

func TestClient_GetVersionForLatest(t *testing.T) {
	c := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar")
	require.NotNil(t, c)

	assert.Panics(t, func() {
		c.GetVersionForLatest("", "default")
	}, "empty OS")

	assert.Panics(t, func() {
		c.GetVersionForLatest("unix", "")
	}, "empty installer type")
}

func TestClient_GetVersionForIp(t *testing.T) {
	c := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar")
	require.NotNil(t, c)

	assert.Panics(t, func() {
		c.GetVersionForIp(nil)
	}, "nil IP")

	assert.Panics(t, func() {
		c.GetVersionForIp(net.IP{})
	}, "empty IP")
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

var goodIp = net.IPv4(192, 168, 0, 1)
var unsetIp = net.IPv4(192, 168, 100, 1)
var unknownIp = net.IPv4(127, 0, 0, 1)

func TestReadVersionForIp(t *testing.T) {
	readFromString := func(ip net.IP, json string) (string, error) {
		r := strings.NewReader(json)
		return readVersionForIp(r, ip)
	}

	{
		v, err := readFromString(goodIp, goodHostsResponse)
		if assert.NoError(t, err) {
			assert.Equal(t, "1.142.0.20180313-173634", v)
		}
	}

	{
		_, err := readFromString(goodIp, "")
		assert.Error(t, err, "empty response")
	}
	{
		_, err := readFromString(unknownIp, "[]")
		assert.Error(t, err, "no hosts")
	}
	{
		_, err := readFromString(unknownIp, goodHostsResponse)
		assert.Error(t, err, "unknown host")
	}
	{
		_, err := readFromString(unsetIp, goodHostsResponse)
		assert.Error(t, err, "no version")
	}
	{
		_, err := readFromString(goodIp, `{"error":{"code":401,"message":"Token Authentication failed"}}`)
		if assert.Error(t, err, "server error") {
			assert.Contains(t, err.Error(), "401")
			assert.Contains(t, err.Error(), "Token Authentication failed")
		}
	}
}
