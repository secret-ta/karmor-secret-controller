// package main

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	securityv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/api/security.kubearmor.com/v1"
// 	karmorfake "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/client/clientset/versioned/fake"
// 	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/runtime"
// 	"k8s.io/client-go/informers"
// 	kubeinformers "k8s.io/client-go/informers"
// 	k8sfake "k8s.io/client-go/kubernetes/fake"
// 	core "k8s.io/client-go/testing"
// 	"k8s.io/client-go/tools/record"
// 	"k8s.io/kubernetes/pkg/apis/apps"
// )

// var (
// 	alwaysReady        = func() bool { return true }
// 	noResyncPeriodFunc = func() time.Duration { return 0 }
// )

// type fixture struct {
// 	t *testing.T

// 	karmorclient *karmorfake.Clientset
// 	kubeclient   *k8sfake.Clientset
// 	// Objects to put in the store.
// 	karmorLister     []*securityv1.KubeArmorPolicy
// 	deploymentLister []*apps.Deployment
// 	// Actions expected to happen on the client.
// 	kubeactions []core.Action
// 	actions     []core.Action
// 	// Objects from here preloaded into NewSimpleFake.
// 	kubeobjects []runtime.Object
// 	objects     []runtime.Object
// }

// func newFixture(t *testing.T) *fixture {
// 	f := &fixture{}
// 	f.t = t
// 	f.objects = []runtime.Object{}
// 	f.kubeobjects = []runtime.Object{}
// 	return f
// }

// func newKarmorPolicy(matchLabels map[string]string, secretPaths []string, secretDirPaths []string, namespace string, policyName string) *securityv1.KubeArmorPolicy {
// 	matchDirectories := []securityv1.FileDirectoryType{}
// 	matchPaths := []securityv1.FilePathType{}

// 	for _, secretDirPath := range secretDirPaths {
// 		matchDirectories = append(matchDirectories, securityv1.FileDirectoryType{
// 			Directory: securityv1.MatchDirectoryType(secretDirPath),
// 			Recursive: true,
// 		})
// 	}

// 	for _, secretPath := range secretPaths {
// 		matchPaths = append(matchPaths, securityv1.FilePathType{
// 			Path: securityv1.MatchPathType(secretPath),
// 		})
// 	}

// 	return &securityv1.KubeArmorPolicy{
// 		ObjectMeta: v1.ObjectMeta{
// 			Name:      policyName,
// 			Namespace: namespace,
// 		},
// 		Spec: securityv1.KubeArmorPolicySpec{
// 			Selector: securityv1.SelectorType{
// 				MatchLabels: matchLabels,
// 			},
// 			File: securityv1.FileType{
// 				MatchPaths:       matchPaths,
// 				MatchDirectories: matchDirectories,
// 			},
// 			Action: securityv1.ActionType("Block"),
// 			Capabilities: securityv1.CapabilitiesType{
// 				MatchCapabilities: []securityv1.MatchCapabilitiesType{},
// 			},
// 			Network: securityv1.NetworkType{
// 				MatchProtocols: []securityv1.MatchNetworkProtocolType{},
// 			},
// 		},
// 	}
// }

// func (f *fixture) newController(ctx context.Context) (*Controller, informers.SharedInformerFactory, kubeinformers.SharedInformerFactory) {
// 	f.karmorclient = karmorfake.NewSimpleClientset(f.objects...)
// 	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

// 	i := informers.NewSharedInformerFactory(f.karmorclient, noResyncPeriodFunc())
// 	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())

// 	c := NewController(ctx, f.kubeclient, f.client,
// 		k8sI.Apps().V1().Deployments(), i.Samplecontroller().V1alpha1().Foos())

// 	c.foosSynced = alwaysReady
// 	c.deploymentsSynced = alwaysReady
// 	c.recorder = &record.FakeRecorder{}

// 	for _, f := range f.fooLister {
// 		i.Samplecontroller().V1alpha1().Foos().Informer().GetIndexer().Add(f)
// 	}

// 	for _, d := range f.deploymentLister {
// 		k8sI.Apps().V1().Deployments().Informer().GetIndexer().Add(d)
// 	}

//		return c, i, k8sI
//	}
package main
