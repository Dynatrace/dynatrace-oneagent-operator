package server

import (
	"net/url"

	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/utils"
	dtwebhook "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/webhook"
)

func getFlavor(libc string /*oa.Spec.LibC*/, annotations map[string]string /* pod.Annotations */) string {
	flavor := url.QueryEscape(utils.GetField(annotations, dtwebhook.AnnotationFlavor, libc))
	if flavor != "musl" {
		return "default"
	}
	return flavor
}
