package main

import (
	"flag"
	"kadeksuryam/secret-controller/pkg/signals"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	karmorclientset "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/client/clientset/versioned"
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

	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	karmorClient, err := karmorclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building karmor clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	controller := NewController(kubeClient, karmorClient, kubeInformerFactory.Apps().V1().Deployments())

	kubeInformerFactory.Start(stopCh)

	if err = controller.Run(int(numworkers), stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.UintVar(&numworkers, "numworker", 2, "Number worker in the controller")
}
