package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	pb "karmor-secret-controller/protobuf"

	"github.com/labstack/gommon/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

func getKubernetesClient() kubernetes.Interface {
	// construct the path to resolve to `~/.kube/config`
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"

	// create the config from the path
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		log.Fatalf("getClusterConfig: %v", err)
	}

	// generate the client based off of the config
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("getClusterConfig: %v", err)
	}

	log.Info("Successfully constructed k8s client")
	return client
}

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
)

var processorClient pb.ProcessorClient

func main() {
	flag.Parse()
	// connect to gRPC server
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	processorClient = pb.NewProcessorClient(conn)

	// connect to k8s cluster
	clientSet := getKubernetesClient()

	// stop signal for the informer
	stopper := make(chan struct{})
	defer close(stopper)

	// create shared informers for resources in all known API group versions with a reSync period and namespace
	factory := informers.NewSharedInformerFactoryWithOptions(clientSet, 10*time.Second, informers.WithNamespace("default"))
	deploymentInformer := factory.Apps().V1().Deployments().Informer()

	defer runtime.HandleCrash()

	// start informer ->
	go factory.Start(stopper)
	// start to sync and call list
	if !cache.WaitForCacheSync(stopper, deploymentInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onAdd, // register add eventhandler
		UpdateFunc: onUpdate,
		DeleteFunc: onDelete,
	})

	// block the main go routine from exiting
	<-stopper
}

func sendRequest(req *pb.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := processorClient.Process(ctx, req)
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
}

func onAdd(obj interface{}) {
	deployment := obj.(*appsv1.Deployment)
	klog.Infof("Deployment CREATED: %s/%s", deployment.Namespace, deployment.Name)
	// go sendRequest(&pb.Request{Namespace: deployment.Namespace, Labels: deployment.Spec.Template.Labels, Action: "PENDING"})
}

func onUpdate(oldObj interface{}, newObj interface{}) {
	oldDeployment := oldObj.(*appsv1.Deployment)
	newDeployment := newObj.(*appsv1.Deployment)
	klog.Infof(
		"Deployment UPDATED. %s/%s %v",
		oldDeployment.Namespace, newDeployment.Name, oldDeployment.Status.AvailableReplicas,
	)
	if newDeployment.Status.ReadyReplicas != newDeployment.Status.Replicas {
		go sendRequest(&pb.Request{Namespace: newDeployment.Namespace, DeploymentName: newDeployment.Name,
			Labels: newDeployment.Spec.Template.Labels, SecretPath: "", Action: "DELETE"})
	} else {
		secretPath := ""
		for _, volume := range newDeployment.Spec.Template.Spec.Volumes {
			if volume.Secret != nil {
				for _, volumeMount := range newDeployment.Spec.Template.Spec.Containers[0].VolumeMounts {
					if volumeMount.Name == volume.Name {
						secretPath = volumeMount.MountPath
					}
				}
			}
		}

		if secretPath != "" {
			log.Print(secretPath)
			go sendRequest(&pb.Request{Namespace: newDeployment.Namespace, DeploymentName: newDeployment.Name,
				Labels: newDeployment.Spec.Template.Labels, SecretPath: secretPath, Action: "CREATE"})
		}
	}
}

func onDelete(obj interface{}) {
	deployment := obj.(*appsv1.Deployment)
	klog.Infof("Deployment DELETED: %s/%s", deployment.Namespace, deployment.Name)
	go sendRequest(&pb.Request{Namespace: deployment.Namespace, DeploymentName: deployment.Name,
		Labels: deployment.Spec.Template.Labels, Action: "DELETE"})
}
