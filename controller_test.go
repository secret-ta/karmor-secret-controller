package main

import (
	"testing"
	"time"

	securityv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/api/security.kubearmor.com/v1"
	"github.com/kubearmor/KubeArmor/pkg/KubeArmorPolicy/client/clientset/versioned/fake"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/kubernetes/pkg/apis/apps"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t *testing.T

	client             *fake.Clientset
	kubeclient         *k8sfake.Clientset
	karmorpolicyLister []*securityv1.KubeArmorPolicy
	deploymentLister   []*apps.Deployment
	kubeactions        []core.Action
	actions            []core.Action
	kubeobjects        []runtime.Object
	objects            []runtime.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.objects = []runtime.Object{}
	f.kubeobjects = []runtime.Object{}

	return f
}
