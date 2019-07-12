package watch

import (
	"fmt"

	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NodeWatcher struct
type NodeWatcher struct {
	kubernetes      kubernetes.Interface
	dynatraceClient dtclient.Client
	logger          logr.Logger

	cordonedNodes map[*v1.Node]bool
}

// NewNodeWatcher - initialises new instance of NodeWatcher
func NewNodeWatcher(
	kubernetes kubernetes.Interface,
	dynatraceClient dtclient.Client,
	logger logr.Logger) *NodeWatcher {

	return &NodeWatcher{
		kubernetes:      kubernetes,
		dynatraceClient: dynatraceClient,
		logger:          logger,
		cordonedNodes:   make(map[*v1.Node]bool),
	}
}

// Watch - this function watches k8s env for any changes in nodes
// only reports to the api when node is found unschedulable
func (nw *NodeWatcher) Watch() {
	api := nw.kubernetes.CoreV1()

	nodes, err := api.Nodes().List(metav1.ListOptions{})
	if err != nil {
		nw.logger.Error(err, "nodewatcher: error listing nodes")
	}
	nw.printNodes(nodes)

	watcher, err := api.Nodes().Watch(metav1.ListOptions{})
	if err != nil {
		nw.logger.Error(err, "nodewatcher: error initialising nodes watcher")
	}

	ch := watcher.ResultChan()
	for event := range ch {

		node, ok := event.Object.(*v1.Node)
		if !ok {
			nw.logger.Error(err, "nodewatcher: error unexpected type")
		}

		if node.Spec.Unschedulable {
			reported, ok := nw.cordonedNodes[node]
			if !ok {
				nw.cordonedNodes[node] = bool(false)
			}
			if !reported {
				nw.sendNodeMarkedForTermination(node)
			}
		}
	}
}

func (nw *NodeWatcher) printNodes(nodes *v1.NodeList) {
	if len(nodes.Items) == 0 {
		err := fmt.Errorf("no items in nodes list")
		nw.logger.Error(err, "printNodes: error listing nodes")
		return
	}
	template := "%-32s%-8s%-8s\n"
	fmt.Println("--- NODES ----")
	fmt.Printf(template, "NAME", "STATUS")
	for _, node := range nodes.Items {
		fmt.Printf(template, node.Name, string(node.Status.Phase))
	}
	fmt.Println("-----------------------------")
}

func (nw *NodeWatcher) sendNodeMarkedForTermination(node *v1.Node) {
	resp, err := nw.dynatraceClient.PostMarkForTerminationEvent(node.GetName())
	if err != nil {
		nw.logger.Error(err, "sendNodeMarkedForTermination: error sending event")
		return
	}
	nw.logger.Info("sendNodeMarkedForTermination: event sent, status %s", resp.Status)
	defer resp.Body.Close()

	nw.cordonedNodes[node] = bool(true)
}
