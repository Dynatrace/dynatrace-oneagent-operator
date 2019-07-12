package watch

import (
	"errors"
	"os"
	"testing"

	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	v1 "k8s.io/api/core/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// TODO
func TestNodeWatcher(t *testing.T) {

	var node = &v1.Node{}
	var fakeInterface = testclient.NewSimpleClientset(node)
	var log = logf.ZapLoggerTo(os.Stdout, true)
	var dtc = new(dtclient.MockDynatraceClient)
	dtc.On("GetVersionForIp", "127.0.0.1").Return("1.2.3", nil)
	dtc.On("GetVersionForIp", "127.0.0.2").Return("0.1.2", nil)
	dtc.On("GetVersionForIp", "127.0.0.3").Return("", errors.New("n/a"))

	_ = NewNodeWatcher(fakeInterface, dtc, log)
	// nw.Watch()
}
