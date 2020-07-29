package nodes

import (
	"context"
	"os"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const cacheName = "dynatrace-node-cache"

type ReconcileNodes struct {
	namespace    string
	client       client.Client
	cache        cache.Cache
	scheme       *runtime.Scheme
	logger       logr.Logger
	dtClientFunc utils.DynatraceClientFunc
	local        bool
}

// Add creates a new Nodes Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	ns, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
	}

	return mgr.Add(&ReconcileNodes{
		namespace:    ns,
		client:       mgr.GetClient(),
		cache:        mgr.GetCache(),
		scheme:       mgr.GetScheme(),
		logger:       log.Log.WithName("nodes.controller"),
		dtClientFunc: utils.BuildDynatraceClient,
		local:        os.Getenv(k8sutil.ForceRunModeEnv) == string(k8sutil.LocalRunMode),
	})
}

// Start starts the Nodes Reconciler, and will block until a stop signal is sent.
func (r *ReconcileNodes) Start(stop <-chan struct{}) error {
	r.cache.WaitForCacheSync(stop)

	chDels, err := r.watchDeletions(stop)
	if err != nil {
		// I've seen watchDeletions() fail because the Cache Informers weren't ready. WaitForCacheSync()
		// should block until they are, however, but I believe I saw this not being true once.
		//
		// Start() failing would exit the Operator process. Since this is a minor feature, let's disable
		// for now until further investigation is done.
		r.logger.Info("failed to initialize watcher for deleted nodes - disabled", "error", err)
		chDels = make(chan string)
	}
	chUpdates, err := r.watchUpdates()
	if err != nil {
		r.logger.Info("failed to initialize watcher for updating nodes - disabled", "error", err)
		chUpdates = make(chan map[string]string)
	}

	chAll := watchTicks(stop, 5*time.Minute)

	for {
		select {
		case <-stop:
			r.logger.Info("stopping nodes controller")
			return nil
		case node := <-chDels:
			if err := r.onDeletion(node); err != nil {
				r.logger.Error(err, "failed to reconcile deletion", "node", node)
			}
		case nodeMap := <-chUpdates:
			if err := r.onUpdate(nodeMap); err != nil {
				r.logger.Error(err, "failed to reconcile updates", "nodes", nodeMap)
			}
		case <-chAll:
			if err := r.reconcileAll(); err != nil {
				r.logger.Error(err, "failed to reconcile nodes")
			}
		}
	}
}

func (r *ReconcileNodes) onUpdate(nodeMap map[string]string) error {
	for oldNode, newNode := range nodeMap {
		logger := r.logger.WithValues("old node", oldNode, "new node", newNode)
		logger.Info("node update notification received")

		cache, err := r.getCache()
		if err != nil {
			return err
		}

		err = r.updateNode(cache, oldNode, newNode, r.findOneAgentByName)
		if err != nil {
			return err
		}

		err = r.updateCache(cache)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileNodes) onDeletion(node string) error {
	log := r.logger.WithValues("node", node)

	log.Info("node deletion notification received")

	cache, err := r.getCache()
	if err != nil {
		return err
	}

	if err = r.removeNode(cache, node, func(oaName string) (*dynatracev1alpha1.OneAgent, error) {
		var oa dynatracev1alpha1.OneAgent
		if err := r.client.Get(context.TODO(), client.ObjectKey{Name: oaName, Namespace: r.namespace}, &oa); err != nil {
			return nil, err
		}
		return &oa, nil
	}); err != nil {
		return err
	}

	return r.updateCache(cache)
}

func (r *ReconcileNodes) reconcileAll() error {
	r.logger.Info("reconciling nodes")

	var oaLst dynatracev1alpha1.OneAgentList
	if err := r.client.List(context.TODO(), &oaLst, client.InNamespace(r.namespace)); err != nil {
		return err
	}

	oas := make(map[string]*dynatracev1alpha1.OneAgent, len(oaLst.Items))
	for i := range oaLst.Items {
		oas[oaLst.Items[i].Name] = &oaLst.Items[i]
	}

	cache, err := r.getCache()
	if err != nil {
		return err
	}

	var nodeLst corev1.NodeList
	if err := r.client.List(context.TODO(), &nodeLst); err != nil {
		return err
	}

	nodes := map[string]bool{}
	for i := range nodeLst.Items {
		node := nodeLst.Items[i]
		nodes[node.Name] = true

		// Sometimes Azure does not cordon off nodes before deleting them since they use taints,
		// this case is handled in the update event handler
		if node.Spec.Unschedulable {
			err = r.reconcileUnschedulableNode(&node, cache)
			if err != nil {
				return err
			}
		}
	}

	// Add or update all nodes seen on OneAgent instances to the cache.
	for _, oa := range oas {
		if oa.Status.Instances != nil {
			for node, info := range oa.Status.Instances {
				if _, ok := nodes[node]; !ok {
					continue
				}

				if err := cache.Set(node, CacheEntry{
					Instance:  oa.Name,
					IPAddress: info.IPAddress,
					LastSeen:  time.Now().UTC(),
				}); err != nil {
					return err
				}
			}
		}
	}

	// Notify and remove all nodes on the cache that aren't in the cluster.
	for _, node := range cache.Keys() {
		if _, ok := nodes[node]; ok {
			continue
		}

		if err := r.removeNode(cache, node, func(name string) (*dynatracev1alpha1.OneAgent, error) {
			if oa, ok := oas[name]; ok {
				return oa, nil
			}

			return nil, errors.NewNotFound(schema.GroupResource{
				Group:    oaLst.GroupVersionKind().Group,
				Resource: oaLst.GroupVersionKind().Kind,
			}, name)
		}); err != nil {
			r.logger.Error(err, "failed to remove node", "node", node)
		}
	}

	return r.updateCache(cache)
}

func (r *ReconcileNodes) getCache() (*Cache, error) {
	var cm corev1.ConfigMap

	err := r.client.Get(context.TODO(), client.ObjectKey{Name: cacheName, Namespace: r.namespace}, &cm)
	if err == nil {
		return &Cache{Obj: &cm}, nil
	}

	if errors.IsNotFound(err) {
		r.logger.Info("no cache found, creating")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cacheName,
				Namespace: r.namespace,
			},
			Data: map[string]string{},
		}

		if !r.local { // If running locally, don't set the controller.
			deploy, err := utils.GetDeployment(r.client, r.namespace)
			if err != nil {
				return nil, err
			}

			if err = controllerutil.SetControllerReference(deploy, cm, r.scheme); err != nil {
				return nil, err
			}
		}

		return &Cache{Create: true, Obj: cm}, nil
	}

	return nil, err
}

func (r *ReconcileNodes) updateCache(c *Cache) error {
	if !c.Changed() {
		return nil
	}

	if c.Create {
		return r.client.Create(context.TODO(), c.Obj)
	}

	return r.client.Update(context.TODO(), c.Obj)
}

func (r *ReconcileNodes) removeNode(c *Cache, node string, oaFunc func(name string) (*dynatracev1alpha1.OneAgent, error)) error {
	log := r.logger.WithValues("node", node)

	nodeInfo, err := c.Get(node)
	if err == ErrNotFound {
		log.Info("ignoring uncached node")
		return nil
	} else if err != nil {
		return err
	}

	if time.Now().UTC().Sub(nodeInfo.LastSeen).Hours() > 1 {
		log.Info("removing stale node")
	} else if nodeInfo.IPAddress == "" {
		log.Info("removing node with unknown IP")
	} else {
		oa, err := oaFunc(nodeInfo.Instance)
		if errors.IsNotFound(err) {
			log.Info("oneagent got already deleted")
			c.Delete(node)
			return nil
		}
		if err != nil {
			return err
		}

		log.Info("sending mark for termination event to dynatrace server", "ip", nodeInfo.IPAddress)

		if err = r.sendMarkedForTermination(oa, nodeInfo.IPAddress, nodeInfo.LastSeen); err != nil {
			return err
		}
	}

	c.Delete(node)
	return nil
}
func (r *ReconcileNodes) updateNode(cache *Cache, oldNode string, newNode string, oaFunc func(name string) (*dynatracev1alpha1.OneAgent, error)) error {
	logger := r.logger

	// If name changes, move cache entry
	if oldNode != newNode {
		oldEntry, err := cache.Get(oldNode)
		if err != nil {
			logger.Error(err, err.Error())
		}

		err = cache.Set(newNode, oldEntry)
		if err != nil {
			logger.Error(err, err.Error())
		}

		cache.Delete(oldNode)
	}

	newNodeInstance := &corev1.Node{}
	err := r.client.Get(context.TODO(), client.ObjectKey{Name: newNode}, newNodeInstance)
	if err != nil {
		return err
	}

	if newNodeInstance.Spec.Taints != nil {
		for _, taint := range newNodeInstance.Spec.Taints {
			if taint.Key == "ToBeDeletedByClusterAutoscaler" {
				// Node is unschedulable if it got taint "ToBeDeletedByClusterAutoscaler" from Azure
				logger.Info("node will be marked for termination because of taint", "node", newNode, "taint", taint.Key)
				cacheEntry, err := cache.Get(newNode)
				if err != nil {
					return err
				}

				lastMarked := cacheEntry.LastMarkedForTermination
				// If the last mark was an hour ago, mark again
				// Zero value for time.Time is 0001-01-01, so first mark is also executed
				if lastMarked.Add(time.Hour).Before(time.Now().UTC()) {
					err = r.reconcileUnschedulableNode(newNodeInstance, cache)
					if err != nil {
						return err
					}
					// After node has been reconciled, loop can be exited
					break
				}
			}
		}
	}

	return nil
}

func (r *ReconcileNodes) sendMarkedForTermination(oa *dynatracev1alpha1.OneAgent, nodeIP string, lastSeen time.Time) error {
	dtc, err := r.dtClientFunc(r.client, oa)
	if err != nil {
		return err
	}

	entityID, err := dtc.GetEntityIDForIP(nodeIP)
	if err != nil {
		return err
	}

	ts := uint64(lastSeen.Add(-10*time.Minute).UnixNano()) / uint64(time.Millisecond)
	return dtc.SendEvent(&dtclient.EventData{
		EventType:     dtclient.MarkedForTerminationEvent,
		Source:        "OneAgent Operator",
		Description:   "Kubernetes node cordoned. Node might be drained or terminated.",
		StartInMillis: ts,
		EndInMillis:   ts,
		AttachRules: dtclient.EventDataAttachRules{
			EntityIDs: []string{entityID},
		},
	})
}

func (r *ReconcileNodes) reconcileUnschedulableNode(node *corev1.Node, cache *Cache) error {
	oneAgent, err := r.determineOneAgentForNode(node.Name)
	if err != nil {
		return err
	}
	if oneAgent == nil {
		return nil
	}

	instance, hasNodeInstance := oneAgent.Status.Instances[node.Name]
	if !hasNodeInstance {
		r.logger.Info("OneAgent has no instance of node", "name", node.Name)
		return nil
	}

	cachedNode, err := cache.Get(node.Name)
	if err != nil {
		return err
	}

	// Update timestamp of last mark
	cachedNode.LastMarkedForTermination = time.Now().UTC()
	err = cache.Set(node.Name, cachedNode)
	if err != nil {
		return err
	}
	return r.sendMarkedForTermination(oneAgent, instance.IPAddress, cachedNode.LastSeen)
}
