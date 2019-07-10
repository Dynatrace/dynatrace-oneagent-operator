package watch

import (
	"testing"

	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestNodeWatcher(t *testing.T) {
	clientset := testclient.NewSimpleClientset()
	serverset := testserver.NewSimpleServerset()
}
