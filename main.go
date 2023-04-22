package main

import (
	"flag"
	"kadeksuryam/secret-controller/pkg/signals"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	karmorclientset "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/client/clientset/versioned"
	karmorpolicyinformer "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/client/informers/externalversions"
	kubeinformers "k8s.io/client-go/informers"
)

var (
	masterURL  string
	kubeconfig string
	numworkers uint
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	ctx := signals.SetupSignalHandler()
	logger := klog.FromContext(ctx)

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		logger.Error(err, "Error building kubeconfig")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	karmorClient, err := karmorclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building karmor clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	karmorpolicyInformerFactory := karmorpolicyinformer.NewSharedInformerFactory(karmorClient, time.Second*30)
	controller := NewController(ctx, kubeClient, karmorClient, kubeInformerFactory.Apps().V1().Deployments(),
		kubeInformerFactory.Apps().V1().StatefulSets(),
		kubeInformerFactory.Apps().V1().DaemonSets(),
		karmorpolicyInformerFactory.Security().V1().KubeArmorPolicies())
	kubeInformerFactory.Start(ctx.Done())
	karmorpolicyInformerFactory.Start(ctx.Done())

	if err = controller.Run(ctx, int(numworkers)); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.UintVar(&numworkers, "numworker", 2, "Number worker in the controller")
}
