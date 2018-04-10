package dynatrace_client

import (
	"net"
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
	c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar")
	require.NoError(t, err)
	require.NotNil(t, c)

	{
		_, err = c.GetVersionForIp(nil)
		assert.Error(t, err, "nil IP")
	}
	{
		_, err = c.GetVersionForIp(net.IP{})
		assert.Error(t, err, "empty IP")
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
