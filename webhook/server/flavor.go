package server

import (
	"net/url"

	"github.com/Dynatrace/dynatrace-oneagent-operator/controllers/utils"
	dtwebhook "github.com/Dynatrace/dynatrace-oneagent-operator/webhook"
)

func getFlavor(libc string, annotations map[string]string) string {
	flavor := url.QueryEscape(utils.GetField(annotations, dtwebhook.AnnotationFlavor, libc))
	if flavor == "musl" {
		return flavor
	}

	return "default"
}
