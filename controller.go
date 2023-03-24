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
	karmorpolicylister karmorpolicylister.KubeArmorPolicyLister
	karmorpolicySynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

func NewController(
	kubeclientset kubernetes.Interface,
	karmorpolicyclientset karmorpolicyclientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	karmorpolicyInformer karmorpolicyinformer.KubeArmorPolicyInformer) *Controller {

	dc := &Controller{
		kubeclientset:         kubeclientset,
		karmorpolicyclientset: karmorpolicyclientset,
		deploymentsLister:     deploymentInformer.Lister(),
		deploymentsSynced:     deploymentInformer.Informer().HasSynced,
		karmorpolicylister:    karmorpolicyInformer.Lister(),
		karmorpolicySynced:    karmorpolicyInformer.Informer().HasSynced,
		workqueue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "karmor-deployment"),
	}

	klog.Info("Setting up event handlers")
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			dc.addDeployment(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			dc.updateDeployment(oldObj, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			dc.deleteDeployment(obj)
		},
	})

	return dc
}

func (dc *Controller) addDeployment(obj interface{}) {
	d := obj.(*appsv1.Deployment)
	klog.Info("Adding deployment", "deployment", klog.KObj(d))
	dc.enqueueDeployment(d)
}

func (dc *Controller) updateDeployment(old, cur interface{}) {
	oldD := old.(*appsv1.Deployment)
	curD := cur.(*appsv1.Deployment)

	if curD.ResourceVersion == oldD.ResourceVersion {
		return
	}

	klog.Info("Updating deployment", "deployment", klog.KObj(oldD))
	dc.enqueueDeployment(curD)
}

func (dc *Controller) deleteDeployment(obj interface{}) {
	d, ok := obj.(*appsv1.Deployment)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
			return
		}
		d, ok = tombstone.Obj.(*appsv1.Deployment)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a Deployment %#v", obj))
			return
		}
	}
	klog.Info("Deleting deployment", "deployment", klog.KObj(d))
	dc.enqueueDeployment(d)
}

func (dc *Controller) enqueueDeployment(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	dc.workqueue.Add(key)
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

	deployment, err := c.deploymentsLister.Deployments(namespace).Get(name)
	if err != nil {
		return err
	}

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
			policyName := namespace + "-" + name + "-" + "disable-secret-access"
			if _, err := c.karmorpolicylister.KubeArmorPolicies(namespace).Get(policyName); err != nil {

				klog.Infof("Creating policy %v", policyName)

				newPolicy := newKarmorSecretPolicy(deployment.Labels, secretDirPaths, namespace, policyName)
				c.karmorpolicyclientset.SecurityV1().KubeArmorPolicies(namespace).Create(context.TODO(), newPolicy, v1.CreateOptions{})
			}
		}
	} else {
		policyName := namespace + "-" + name + "-" + "disable-secret-access"
		if _, err := c.karmorpolicylister.KubeArmorPolicies(namespace).Get(policyName); err == nil {
			klog.Infof("Deleting policy %v", policyName)
			c.karmorpolicyclientset.SecurityV1().KubeArmorPolicies(namespace).Delete(context.TODO(), policyName, v1.DeleteOptions{})
		}
	}
	return nil
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
