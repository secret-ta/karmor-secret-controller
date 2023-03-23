package main

import (
	"context"
	"fmt"
	"time"

	securityv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/api/security.kubearmor.com/v1"
	karmorclientset "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/client/clientset/versioned"
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
	kubeclientset   kubernetes.Interface
	karmorclientset karmorclientset.Interface

	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
	workqueue         workqueue.RateLimitingInterface
}

func NewController(
	kubeclientset kubernetes.Interface,
	karmorclientset karmorclientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer) *Controller {

	dc := &Controller{
		kubeclientset:     kubeclientset,
		karmorclientset:   karmorclientset,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "karmor-deployment"),
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
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced); !ok {
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
		secretPath := ""
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.Secret != nil {
				for _, volumeMount := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
					if volumeMount.Name == volume.Name {
						secretPath = volumeMount.MountPath
					}
				}
			}
		}

		if secretPath != "" && namespace == "default" {
			if secretPath[len(secretPath)-1] != '/' {
				secretPath += "/"
			}
			newPolicy := newKarmorSecretPolicy(deployment.Labels, secretPath, namespace, name)
			_, err = c.karmorclientset.SecurityV1().KubeArmorPolicies(namespace).Create(context.TODO(), newPolicy, v1.CreateOptions{})
		}
	} else {
		policyName := namespace + "-" + name + "-" + "disable-secret-access"
		c.karmorclientset.SecurityV1().KubeArmorPolicies(namespace).Delete(context.TODO(), policyName, v1.DeleteOptions{})
	}
	if err != nil {
		policyName := namespace + "-" + name + "-" + "disable-secret-access"
		c.karmorclientset.SecurityV1().KubeArmorPolicies(namespace).Delete(context.TODO(), policyName, v1.DeleteOptions{})
		return err
	}

	return nil
}

func newKarmorSecretPolicy(matchLabels map[string]string, secretDirPath string, namespace string, name string) *securityv1.KubeArmorPolicy {
	policyName := namespace + "-" + name + "-" + "disable-secret-access"
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
				MatchDirectories: []securityv1.FileDirectoryType{
					{
						Directory: securityv1.MatchDirectoryType(secretDirPath),
						Recursive: true,
					},
				},
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
