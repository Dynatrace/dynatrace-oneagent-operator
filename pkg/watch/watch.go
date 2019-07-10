package watch

import (
	"fmt"

	"k8s.io/client-go/rest"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
)

// NodeWatcher watches the nodes of given k8s env
type NodeWatcher struct {
	config *rest.Config
}

// NewNodeWatcher -
// Initialises a new instance of nodewatcher
func NewNodeWatcher(config *rest.Config) *NodeWatcher {
	return &NodeWatcher{
		config: config,
	}
}

// Watch - this function watches k8s env for any changes in nodes
// only reports to the api when node is found unschedulable
func (nw *NodeWatcher) Watch(clientset *kubernetes.Clientset, logger logr.Logger) {

	if nw.config == nil {
		err := fmt.Errorf("config not set")
		logger.Error(err, "nodewatcher: error initialising")
		return
	}

	// clientset, err := kubernetes.NewForConfig(nw.config)
	// if err != nil {
	// 	logger.Error(err, "nodewatcher: error initialising kubernetes client")
	// }

	api := clientset.CoreV1()
	nodes, err := api.Nodes().List(metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "nodewatcher: error listing nodes")
	}
	printNodes(nodes, logger)

	watcher, err := api.Nodes().Watch(metav1.ListOptions{})
	if err != nil {
		// log.Fatal(err)
	}
	ch := watcher.ResultChan()

	for event := range ch {

		node, ok := event.Object.(*v1.Node)
		if !ok {
			// log.Fatal("unexpected type")
		}
		if node.Spec.Unschedulable {
			// log.Printf("node schedulable %v", node.Spec.Unschedulable)
			// log.Printf("node event %v", event)
		}
	}
}

func printNodes(nodes *v1.NodeList, logger logr.Logger) {
	if len(nodes.Items) == 0 {
		err := fmt.Errorf("no items in nodes list")
		logger.Error(err, "printNodes: error listing nodes")
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
