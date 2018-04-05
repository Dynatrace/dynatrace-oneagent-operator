package dynatrace_client

import (
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO test GetVersionForLatest when implemented

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
