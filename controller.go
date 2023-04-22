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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
		klog.Infof("Adding deployment: %v", klog.KObj(v))
	case *appsv1.StatefulSet:
		klog.Infof("Adding statefulset: %v", klog.KObj(v))
	case *appsv1.DaemonSet:
		klog.Infof("Adding daemonset: %v", klog.KObj(v))
	default:
		klog.Error("Unrecognized object type")
		panic(obj)
	}
	c.enqueue(obj)
}

func (c *Controller) update(oldObj, newObj interface{}) {
	switch v := newObj.(type) {
	case *appsv1.Deployment:
		klog.Infof("Updating deployment: %v", klog.KObj(v))
	case *appsv1.StatefulSet:
		klog.Infof("Updating statefulset: %v", klog.KObj(v))
	case *appsv1.DaemonSet:
		klog.Infof("Adding daemonset: %v", klog.KObj(v))
	default:
		klog.Error("Unrecognized object type")
		panic(v)
	}

	c.enqueue(newObj)
}

func (c *Controller) delete(obj interface{}) {
	_, ok := obj.(*appsv1.Deployment)
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
		klog.Infof("Deleting deployment: %v", klog.KObj(v))
	case *appsv1.StatefulSet:
		klog.Infof("Adding statefulset: %v", klog.KObj(v))
	case *appsv1.DaemonSet:
		klog.Infof("Adding daemonset: %v", klog.KObj(v))
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

func getDeploymentKarmorPolicyName(namespace, name string) string {
	return namespace + "-" + name + "-" + "deployment-disable-secret-access"
}

func getStatefulSetPolicyName(namespace, name string) string {
	return namespace + "-" + name + "-" + "stateful-disable-secret-access"
}
func getDaemonSetPolicyName(namespace, name string) string {
	return namespace + "-" + name + "-" + "daemonset-disable-secret-access"
}
func (c *Controller) deleteKarmorPolicy(namespace, name string) error {
	_, err := c.getKarmorPolicy(namespace, name)
	if err == nil {
		err = c.karmorpolicyclientset.SecurityV1().KubeArmorPolicies(namespace).Delete(context.TODO(), name, v1.DeleteOptions{})
	}
	return err
}

func (c *Controller) getKarmorPolicy(namespace, name string) (*securityv1.KubeArmorPolicy, error) {
	return c.karmorpolicylister.KubeArmorPolicies(namespace).Get(name)
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return err
	}

	deployment, deploymentErr := c.deploymentsLister.Deployments(namespace).Get(name)
	statefulset, statefulsetErr := c.statefulSetLister.StatefulSets(namespace).Get(name)
	daemonset, daemonsetErr := c.daemonSetLister.DaemonSets(namespace).Get(name)

	c.handleDeployment(namespace, name, deployment, deploymentErr)
	c.handleStatefulSet(namespace, name, statefulset, statefulsetErr)
	c.handleDaemonSet(namespace, name, daemonset, daemonsetErr)

	return nil
}

func (c *Controller) handleDeployment(namespace, name string, deployment *appsv1.Deployment, err error) {
	if errors.IsNotFound(err) {
		klog.Info("Deployment has been deleted: ", namespace, "/", name)
		c.deleteKarmorPolicy(namespace, getDeploymentKarmorPolicyName(namespace, name))
		return
	}

	_, policyErr := c.getKarmorPolicy(namespace, getDeploymentKarmorPolicyName(namespace, name))
	if !errors.IsNotFound(policyErr) && (deployment.Status.ReadyReplicas == deployment.Status.Replicas) {
		return
	}

	c.processDeploymentWorkload(namespace, name, deployment)
}

func (c *Controller) handleStatefulSet(namespace, name string, statefulset *appsv1.StatefulSet, err error) {
	if errors.IsNotFound(err) {
		klog.Info("StatefulSet has been deleted: ", namespace, "/", name)
		c.deleteKarmorPolicy(namespace, getStatefulSetPolicyName(namespace, name))
		return
	}

	_, policyErr := c.getKarmorPolicy(namespace, getStatefulSetPolicyName(namespace, name))
	if !errors.IsNotFound(policyErr) && (statefulset.Status.ReadyReplicas == statefulset.Status.Replicas) {
		return
	}

	c.processStatefulSetWorkload(namespace, name, statefulset)
}

func (c *Controller) handleDaemonSet(namespace, name string, daemonset *appsv1.DaemonSet, err error) {
	if errors.IsNotFound(err) {
		klog.Info("Daemonset has been deleted: ", namespace, "/", name)
		c.deleteKarmorPolicy(namespace, getDaemonSetPolicyName(namespace, name))
		return
	}

	_, policyErr := c.getKarmorPolicy(namespace, getDaemonSetPolicyName(namespace, name))
	if !errors.IsNotFound(policyErr) && (daemonset.Status.DesiredNumberScheduled == daemonset.Status.NumberAvailable) {
		return
	}

	c.processDaemonSetWorkload(namespace, name, daemonset)
}

func (c *Controller) processWorkload(namespace, name string, template *corev1.PodTemplateSpec, isReady bool, getPolicyName func(string, string) string) {
	if isReady {
		secretPaths := []string{}
		secretDirPaths := []string{}
		labels := template.Labels
		needSecureEnv := false
		for key, value := range labels {
			if key == "env-secret-secured" && value == "true" {
				needSecureEnv = true
				break
			}
		}

		if needSecureEnv {
			secretDirPaths = append(secretDirPaths, "/vol/")
			secretPaths = append(secretPaths, "/proc/1/environ")
		}

		for _, volume := range template.Spec.Volumes {
			if volume.Secret != nil {
				for _, container := range template.Spec.Containers {
					for _, containerVolMount := range container.VolumeMounts {
						if containerVolMount.Name == volume.Name {
							secretDirPath := containerVolMount.MountPath
							if secretDirPath[len(secretDirPath)-1] != '/' {
								secretDirPath += "/"
							}
							secretDirPaths = append(secretDirPaths, secretDirPath)
						}
					}
				}
			}
		}

		if len(secretDirPaths) != 0 {
			policyName := getPolicyName(namespace, name)
			if _, err := c.karmorpolicylister.KubeArmorPolicies(namespace).Get(policyName); err != nil {
				klog.Infof("Creating policy %v", policyName)

				newPolicy := newKarmorSecretPolicy(labels, secretPaths, secretDirPaths, namespace, policyName)
				_, err := c.karmorpolicyclientset.SecurityV1().KubeArmorPolicies(namespace).Create(context.TODO(), newPolicy, v1.CreateOptions{})
				if err != nil {
					klog.Errorf("Error creating policy: %v", err)
				}

			}
		}
	} else {
		c.deleteKarmorPolicy(namespace, getPolicyName(namespace, name))
	}
}

func (c *Controller) processDeploymentWorkload(namespace, name string, deployment *appsv1.Deployment) {
	isReady := deployment.Status.ReadyReplicas == deployment.Status.Replicas
	c.processWorkload(namespace, name, &deployment.Spec.Template, isReady, getDeploymentKarmorPolicyName)
}

func (c *Controller) processStatefulSetWorkload(namespace, name string, statefulset *appsv1.StatefulSet) {
	isReady := statefulset.Status.ReadyReplicas == statefulset.Status.Replicas
	c.processWorkload(namespace, name, &statefulset.Spec.Template, isReady, getStatefulSetPolicyName)
}

func (c *Controller) processDaemonSetWorkload(namespace, name string, daemonset *appsv1.DaemonSet) {
	isReady := daemonset.Status.NumberReady == daemonset.Status.NumberAvailable
	c.processWorkload(namespace, name, &daemonset.Spec.Template, isReady, getDaemonSetPolicyName)
}

func newKarmorSecretPolicy(matchLabels map[string]string, secretPaths []string, secretDirPaths []string, namespace string, policyName string) *securityv1.KubeArmorPolicy {
	matchDirectories := []securityv1.FileDirectoryType{}
	matchPaths := []securityv1.FilePathType{}

	for _, secretDirPath := range secretDirPaths {
		matchDirectories = append(matchDirectories, securityv1.FileDirectoryType{
			Directory: securityv1.MatchDirectoryType(secretDirPath),
			Recursive: true,
		})
	}

	for _, secretPath := range secretPaths {
		matchPaths = append(matchPaths, securityv1.FilePathType{
			Path: securityv1.MatchPathType(secretPath),
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
				MatchPaths:       matchPaths,
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
