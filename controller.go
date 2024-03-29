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
	karmorpolicyLister karmorpolicylister.KubeArmorPolicyLister
	karmorpolicySynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

func NewController(
	ctx context.Context,
	kubeclientset kubernetes.Interface,
	karmorpolicyclientset karmorpolicyclientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	statefulSetInformer appsinformers.StatefulSetInformer,
	daemonSetInformer appsinformers.DaemonSetInformer,
	karmorpolicyInformer karmorpolicyinformer.KubeArmorPolicyInformer) *Controller {
	logger := klog.FromContext(ctx)

	c := &Controller{
		kubeclientset:         kubeclientset,
		karmorpolicyclientset: karmorpolicyclientset,
		deploymentsLister:     deploymentInformer.Lister(),
		deploymentsSynced:     deploymentInformer.Informer().HasSynced,
		statefulSetLister:     statefulSetInformer.Lister(),
		statefulSetSynced:     statefulSetInformer.Informer().HasSynced,
		daemonSetLister:       daemonSetInformer.Lister(),
		daemonSetSynced:       daemonSetInformer.Informer().HasSynced,
		karmorpolicyLister:    karmorpolicyInformer.Lister(),
		karmorpolicySynced:    karmorpolicyInformer.Informer().HasSynced,
		workqueue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "karmor-secret"),
	}

	logger.Info("Setting up event handlers")
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.add(ctx, obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.update(ctx, oldObj, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			c.delete(ctx, obj)
		},
	})

	statefulSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.add(ctx, obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.update(ctx, oldObj, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			c.delete(ctx, obj)
		},
	})

	daemonSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.add(ctx, obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.update(ctx, oldObj, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			c.delete(ctx, obj)
		},
	})

	return c
}

func (c *Controller) add(ctx context.Context, obj interface{}) {
	switch v := obj.(type) {
	case *appsv1.Deployment:
	case *appsv1.StatefulSet:
	case *appsv1.DaemonSet:
	default:
		panic(v)
	}
	c.enqueue(obj)
}

func (c *Controller) update(ctx context.Context, oldObj, newObj interface{}) {
	switch v := newObj.(type) {
	case *appsv1.Deployment:
	case *appsv1.StatefulSet:
	case *appsv1.DaemonSet:
	default:
		panic(v)
	}

	c.enqueue(newObj)
}

func (c *Controller) delete(ctx context.Context, obj interface{}) {
	switch v := obj.(type) {
	case *appsv1.Deployment:
	case *appsv1.StatefulSet:
	case *appsv1.DaemonSet:
	default:
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

func (c *Controller) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	logger := klog.FromContext(ctx)

	logger.Info("Starting karmor secret controller")

	logger.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.deploymentsSynced, c.karmorpolicySynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	logger.Info("Starting workers")
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	logger.Info("Started workers")
	<-ctx.Done()
	klog.Info("Shutting down workers")

	return nil
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
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

		if err := c.syncHandler(ctx, key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(obj)
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
	return c.karmorpolicyLister.KubeArmorPolicies(namespace).Get(name)
}

func (c *Controller) syncHandler(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return err
	}

	deployment, deploymentErr := c.deploymentsLister.Deployments(namespace).Get(name)
	statefulset, statefulsetErr := c.statefulSetLister.StatefulSets(namespace).Get(name)
	daemonset, daemonsetErr := c.daemonSetLister.DaemonSets(namespace).Get(name)

	c.handleDeployment(ctx, namespace, name, deployment, deploymentErr)
	c.handleStatefulSet(ctx, namespace, name, statefulset, statefulsetErr)
	c.handleDaemonSet(ctx, namespace, name, daemonset, daemonsetErr)

	return nil
}

func (c *Controller) handleDeployment(ctx context.Context, namespace, name string, deployment *appsv1.Deployment, err error) {
	policyName := getDeploymentKarmorPolicyName(namespace, name)

	if errors.IsNotFound(err) {
		c.deleteKarmorPolicy(namespace, policyName)
		return
	}

	isNotReady := deployment.Status.ReadyReplicas != deployment.Status.Replicas
	if isNotReady {
		c.deleteKarmorPolicy(namespace, policyName)
		return
	}

	_, policyErr := c.getKarmorPolicy(namespace, policyName)
	if !errors.IsNotFound(policyErr) {
		return
	}

	c.processDeploymentWorkload(ctx, namespace, name, deployment)
}

func (c *Controller) handleStatefulSet(ctx context.Context, namespace, name string, statefulset *appsv1.StatefulSet, err error) {
	policyName := getStatefulSetPolicyName(namespace, name)

	if errors.IsNotFound(err) {
		c.deleteKarmorPolicy(namespace, policyName)
		return
	}

	isNotReady := statefulset.Status.ReadyReplicas != statefulset.Status.Replicas
	if isNotReady {
		c.deleteKarmorPolicy(namespace, policyName)
		return
	}

	_, policyErr := c.getKarmorPolicy(namespace, policyName)
	if !errors.IsNotFound(policyErr) {
		return
	}

	c.processStatefulSetWorkload(ctx, namespace, name, statefulset)
}

func (c *Controller) handleDaemonSet(ctx context.Context, namespace, name string, daemonset *appsv1.DaemonSet, err error) {
	policyName := getDaemonSetPolicyName(namespace, name)

	if errors.IsNotFound(err) {
		c.deleteKarmorPolicy(namespace, policyName)
		return
	}

	isNotReady := daemonset.Status.DesiredNumberScheduled != daemonset.Status.NumberAvailable
	if isNotReady {
		c.deleteKarmorPolicy(namespace, policyName)
		return
	}

	_, policyErr := c.getKarmorPolicy(namespace, policyName)
	if !errors.IsNotFound(policyErr) {
		return
	}

	c.processDaemonSetWorkload(ctx, namespace, name, daemonset)
}

func (c *Controller) processWorkload(ctx context.Context, namespace, name string, template *corev1.PodTemplateSpec, getPolicyName func(string, string) string) {
	logger := klog.FromContext(ctx)

	secretPaths := []string{}
	secretDirPaths := []string{}
	labels := template.Labels
	needSecureEnv := false
	needSecureMountedFile := false

	for key, value := range labels {
		if key == "env-secret-secured" && value == "true" {
			needSecureEnv = true
		}
		if key == "mf-secret-secured" && value == "true" {
			needSecureMountedFile = true
		}
	}

	if !needSecureEnv && !needSecureMountedFile {
		return
	}

	if needSecureEnv {
		secretDirPaths = append(secretDirPaths, "/vol/")
		secretPaths = append(secretPaths, "/proc/1/environ")
	}

	if needSecureMountedFile {
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
	}

	policyName := getPolicyName(namespace, name)
	logger.Info("Creating policy", policyName)

	newPolicy := newKarmorSecretPolicy(labels, secretPaths, secretDirPaths, namespace, policyName)
	_, err := c.karmorpolicyclientset.SecurityV1().KubeArmorPolicies(namespace).Create(context.TODO(), newPolicy, v1.CreateOptions{})
	if err != nil {
		logger.Error(err, "Error creating policy")
	}
}

func (c *Controller) processDeploymentWorkload(ctx context.Context, namespace, name string, deployment *appsv1.Deployment) {
	c.processWorkload(ctx, namespace, name, &deployment.Spec.Template, getDeploymentKarmorPolicyName)
}

func (c *Controller) processStatefulSetWorkload(ctx context.Context, namespace, name string, statefulset *appsv1.StatefulSet) {
	c.processWorkload(ctx, namespace, name, &statefulset.Spec.Template, getStatefulSetPolicyName)
}

func (c *Controller) processDaemonSetWorkload(ctx context.Context, namespace, name string, daemonset *appsv1.DaemonSet) {
	c.processWorkload(ctx, namespace, name, &daemonset.Spec.Template, getDaemonSetPolicyName)
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
