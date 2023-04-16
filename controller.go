package main

import (
	"context"
	"fmt"
	"time"

	securityv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/api/security.kubearmor.com/v1"
	karmorpolicyclientset "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/client/clientset/versioned"
	karmorpolicyinformer "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/client/informers/externalversions/security.kubearmor.com/v1"
	karmorpolicylister "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/client/listers/security.kubearmor.com/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type Controller struct {
	kubeclientset         kubernetes.Interface
	karmorpolicyclientset karmorpolicyclientset.Interface

	deploymentsLister  appslisters.DeploymentLister
	deploymentsSynced  cache.InformerSynced
	statefulSetLister  appslisters.StatefulSetLister
	statefulSetSynced  cache.InformerSynced
	daemonSetLister    appslisters.DaemonSetLister
	daemonSetSynced    cache.InformerSynced
	karmorpolicylister karmorpolicylister.KubeArmorPolicyLister
	karmorpolicySynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

func NewController(
	kubeclientset kubernetes.Interface,
	karmorpolicyclientset karmorpolicyclientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	statefulSetInformer appsinformers.StatefulSetInformer,
	daemonSetInformer appsinformers.DaemonSetInformer,
	karmorpolicyInformer karmorpolicyinformer.KubeArmorPolicyInformer) *Controller {

	c := &Controller{
		kubeclientset:         kubeclientset,
		karmorpolicyclientset: karmorpolicyclientset,
		deploymentsLister:     deploymentInformer.Lister(),
		deploymentsSynced:     deploymentInformer.Informer().HasSynced,
		statefulSetLister:     statefulSetInformer.Lister(),
		statefulSetSynced:     statefulSetInformer.Informer().HasSynced,
		daemonSetLister:       daemonSetInformer.Lister(),
		daemonSetSynced:       daemonSetInformer.Informer().HasSynced,
		karmorpolicylister:    karmorpolicyInformer.Lister(),
		karmorpolicySynced:    karmorpolicyInformer.Informer().HasSynced,
		workqueue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "karmor-secret"),
	}

	klog.Info("Setting up event handlers")
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.add(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.update(oldObj, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			c.delete(obj)
		},
	})

	statefulSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.add(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.update(oldObj, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			c.delete(obj)
		},
	})

	daemonSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.add(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.update(oldObj, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			c.delete(obj)
		},
	})

	return c
}

func (c *Controller) add(obj interface{}) {
	switch v := obj.(type) {
	case *appsv1.Deployment:
		klog.Info("Adding deployment", "deployment", klog.KObj(v))
	case *appsv1.StatefulSet:
		klog.Info("Adding statefulset", "statefulset", klog.KObj(v))
	case *appsv1.DaemonSet:
		klog.Info("Adding daemonset", "daemonset", klog.KObj(v))
	default:
		klog.Error("Unrecognized object type")
		panic(obj)
	}
	c.enqueue(obj)
}

func (c *Controller) update(old, cur interface{}) {
	oldObj := old.(v1.Object)
	curObj := old.(v1.Object)

	switch v := cur.(type) {
	case *appsv1.Deployment:
		klog.Info("Updating deployment", "deployment", klog.KObj(v))
	case *appsv1.StatefulSet:
		klog.Info("Updating statefulset", "statefulset", klog.KObj(v))
	case *appsv1.DaemonSet:
		klog.Info("Updating daemonset", "daemonset", klog.KObj(v))
	default:
		klog.Error("Unrecognized object type")
		panic(v)
	}

	if oldObj.GetResourceVersion() == curObj.GetResourceVersion() {
		return
	}
	c.enqueue(cur)
}

func (c *Controller) delete(obj interface{}) {
	_, ok := obj.(v1.Object)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
			return
		}
		_, ok = tombstone.Obj.(*appsv1.Deployment)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a Deployment %#v", obj))
			return
		}
	}
	switch v := obj.(type) {
	case *appsv1.Deployment:
		klog.Info("Deleting deployment", "deployment", klog.KObj(v))
	case *appsv1.StatefulSet:
		klog.Info("Deleting statefulset", "statefulset", klog.KObj(v))
	case *appsv1.DaemonSet:
		klog.Info("Deleting daemonset", "daemonset", klog.KObj(v))
	default:
		klog.Error("Unrecognized object type")
		panic(v)
	}
	c.enqueue(obj)
}

func (c *Controller) enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("Starting karmor secret controller")

	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.karmorpolicySynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	klog.Info("Starting workers")
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	fmt.Println("err, obj: ", obj)

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	deployment, deploymentErr := c.deploymentsLister.Deployments(namespace).Get(name)
	statefulset, statefulsetErr := c.statefulSetLister.StatefulSets(namespace).Get(name)
	daemonset, daemonsetErr := c.daemonSetLister.DaemonSets(namespace).Get(name)

	if deploymentErr == nil {
		c.processDeploymentWorkload(namespace, name, deployment)
	} else if statefulsetErr == nil {
		c.processStatefulSetWorkload(namespace, name, statefulset)
	} else if daemonsetErr == nil {
		c.processDaemonSetWorkload(namespace, name, daemonset)
	} else {
		panic(key)
	}

	return nil
}

func (c *Controller) processDeploymentWorkload(namespace, name string, deployment *appsv1.Deployment) {
	klog.Infof("Process deployment: %v", deployment.Name)
	if deployment.Status.ReadyReplicas == deployment.Status.Replicas {
		secretPaths := map[string]struct{}{}
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.Secret != nil {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					for _, containerVolMount := range container.VolumeMounts {
						if containerVolMount.Name == volume.Name {
							secretPaths[containerVolMount.MountPath] = struct{}{}
						}
					}
				}
			}
		}

		secretDirPaths := []string{}
		for secretPath := range secretPaths {
			if secretPath[len(secretPath)-1] != '/' {
				secretPath += "/"
			}
			secretDirPaths = append(secretDirPaths, secretPath)
		}

		if len(secretDirPaths) != 0 {
			policyName := namespace + "-" + name + "-" + "deployment-disable-secret-access"
			if _, err := c.karmorpolicylister.KubeArmorPolicies(namespace).Get(policyName); err != nil {

				klog.Infof("Creating policy %v", policyName)

				newPolicy := newKarmorSecretPolicy(deployment.Labels, secretDirPaths, namespace, policyName)
				c.karmorpolicyclientset.SecurityV1().KubeArmorPolicies(namespace).Create(context.TODO(), newPolicy, v1.CreateOptions{})
			}
		}
	} else {
		policyName := namespace + "-" + name + "-" + "deployment-disable-secret-access"
		if _, err := c.karmorpolicylister.KubeArmorPolicies(namespace).Get(policyName); err == nil {
			klog.Infof("Deleting policy %v", policyName)
			c.karmorpolicyclientset.SecurityV1().KubeArmorPolicies(namespace).Delete(context.TODO(), policyName, v1.DeleteOptions{})
		}
	}
}

func (c *Controller) processStatefulSetWorkload(namespace, name string, statefulset *appsv1.StatefulSet) {
	klog.Infof("Process statefulset: %v", statefulset.Name)
	if statefulset.Status.ReadyReplicas == statefulset.Status.Replicas {
		secretPaths := map[string]struct{}{}
		for _, volume := range statefulset.Spec.Template.Spec.Volumes {
			if volume.Secret != nil {
				for _, container := range statefulset.Spec.Template.Spec.Containers {
					for _, containerVolMount := range container.VolumeMounts {
						if containerVolMount.Name == volume.Name {
							secretPaths[containerVolMount.MountPath] = struct{}{}
						}
					}
				}
			}
		}

		secretDirPaths := []string{}
		for secretPath := range secretPaths {
			if secretPath[len(secretPath)-1] != '/' {
				secretPath += "/"
			}
			secretDirPaths = append(secretDirPaths, secretPath)
		}

		if len(secretDirPaths) != 0 {
			policyName := namespace + "-" + name + "-" + "statefulset-disable-secret-access"
			if _, err := c.karmorpolicylister.KubeArmorPolicies(namespace).Get(policyName); err != nil {

				klog.Infof("Creating policy %v", policyName)

				newPolicy := newKarmorSecretPolicy(statefulset.Labels, secretDirPaths, namespace, policyName)
				c.karmorpolicyclientset.SecurityV1().KubeArmorPolicies(namespace).Create(context.TODO(), newPolicy, v1.CreateOptions{})
			}
		}
	} else {
		policyName := namespace + "-" + name + "-" + "statefulset-disable-secret-access"
		if _, err := c.karmorpolicylister.KubeArmorPolicies(namespace).Get(policyName); err == nil {
			klog.Infof("Deleting policy %v", policyName)
			c.karmorpolicyclientset.SecurityV1().KubeArmorPolicies(namespace).Delete(context.TODO(), policyName, v1.DeleteOptions{})
		}
	}
}

func (c *Controller) processDaemonSetWorkload(namespace, name string, daemonset *appsv1.DaemonSet) {
	klog.Infof("Process statefulset: %v", daemonset.Name)
	if daemonset.Status.NumberReady == daemonset.Status.NumberAvailable {
		secretPaths := map[string]struct{}{}
		for _, volume := range daemonset.Spec.Template.Spec.Volumes {
			if volume.Secret != nil {
				for _, container := range daemonset.Spec.Template.Spec.Containers {
					for _, containerVolMount := range container.VolumeMounts {
						if containerVolMount.Name == volume.Name {
							secretPaths[containerVolMount.MountPath] = struct{}{}
						}
					}
				}
			}
		}

		secretDirPaths := []string{}
		for secretPath := range secretPaths {
			if secretPath[len(secretPath)-1] != '/' {
				secretPath += "/"
			}
			secretDirPaths = append(secretDirPaths, secretPath)
		}

		if len(secretDirPaths) != 0 {
			policyName := namespace + "-" + name + "-" + "daemonset-disable-secret-access"
			if _, err := c.karmorpolicylister.KubeArmorPolicies(namespace).Get(policyName); err != nil {

				klog.Infof("Creating policy %v", policyName)

				newPolicy := newKarmorSecretPolicy(daemonset.Labels, secretDirPaths, namespace, policyName)
				c.karmorpolicyclientset.SecurityV1().KubeArmorPolicies(namespace).Create(context.TODO(), newPolicy, v1.CreateOptions{})
			}
		}
	} else {
		policyName := namespace + "-" + name + "-" + "daemonset-disable-secret-access"
		if _, err := c.karmorpolicylister.KubeArmorPolicies(namespace).Get(policyName); err == nil {
			klog.Infof("Deleting policy %v", policyName)
			c.karmorpolicyclientset.SecurityV1().KubeArmorPolicies(namespace).Delete(context.TODO(), policyName, v1.DeleteOptions{})
		}
	}
}

func newKarmorSecretPolicy(matchLabels map[string]string, secretDirPaths []string, namespace string, policyName string) *securityv1.KubeArmorPolicy {
	matchDirectories := []securityv1.FileDirectoryType{}

	for _, secretDirPath := range secretDirPaths {
		matchDirectories = append(matchDirectories, securityv1.FileDirectoryType{
			Directory: securityv1.MatchDirectoryType(secretDirPath),
			Recursive: true,
		})
	}

	return &securityv1.KubeArmorPolicy{
		ObjectMeta: v1.ObjectMeta{
			Name:      policyName,
			Namespace: namespace,
		},
		Spec: securityv1.KubeArmorPolicySpec{
			Selector: securityv1.SelectorType{
				MatchLabels: matchLabels,
			},
			File: securityv1.FileType{
				MatchDirectories: matchDirectories,
			},
			Action: securityv1.ActionType("Block"),
			Capabilities: securityv1.CapabilitiesType{
				MatchCapabilities: []securityv1.MatchCapabilitiesType{},
			},
			Network: securityv1.NetworkType{
				MatchProtocols: []securityv1.MatchNetworkProtocolType{},
			},
		},
	}
}
