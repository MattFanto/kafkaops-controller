package main

//import (
//	"fmt"
//	informers "github.com/mattfanto/kafkaops-controller/pkg/generated/informers/externalversions"
//	"k8s.io/apimachinery/pkg/runtime/schema"
//	"k8s.io/apimachinery/pkg/util/diff"
//	kubeinformers "k8s.io/client-go/informers"
//	"k8s.io/client-go/tools/cache"
//	"k8s.io/client-go/tools/record"
//
//	"reflect"
//	"testing"
//	"time"
//
//	kafkaopscontroller "github.com/mattfanto/kafkaops-controller/pkg/apis/kafkaopscontroller/v1alpha1"
//	"github.com/mattfanto/kafkaops-controller/pkg/generated/clientset/versioned/fake"
//	apps "k8s.io/api/apps/v1"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/runtime"
//	k8sfake "k8s.io/client-go/kubernetes/fake"
//	core "k8s.io/client-go/testing"
//)
import (
	"github.com/mattfanto/kafkaops-controller/pkg/resources/kafkaops"
	"reflect"
	"testing"
	"time"

	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/diff"
	kubeinformers "k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	kafkaopscontroller "github.com/mattfanto/kafkaops-controller/pkg/apis/kafkaopscontroller/v1alpha1"
	"github.com/mattfanto/kafkaops-controller/pkg/generated/clientset/versioned/fake"
	informers "github.com/mattfanto/kafkaops-controller/pkg/generated/informers/externalversions"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t *testing.T

	client     *fake.Clientset
	kubeclient *k8sfake.Clientset
	// Objects to put in the store.
	kafkaTopicsLister []*kafkaopscontroller.KafkaTopic
	// Actions expected to happen on the client.
	kubeactions []core.Action
	actions     []core.Action
	// Objects from here preloaded into NewSimpleFake.
	kubeobjects []runtime.Object
	objects     []runtime.Object
	kafkaSdk    kafkaops.Interface
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.objects = []runtime.Object{}
	f.kubeobjects = []runtime.Object{}
	f.kafkaSdk = fakeKafkaSdk{}
	return f
}

func newFoo(name string, replicas *int32, partitions *int32) *kafkaopscontroller.KafkaTopic {
	return &kafkaopscontroller.KafkaTopic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kafkaopscontroller.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: kafkaopscontroller.KafkaTopicSpec{
			TopicName:  name,
			Replicas:   replicas,
			Partitions: partitions,
		},
		Status: kafkaopscontroller.KafkaTopicStatus{
			StatusCode: kafkaopscontroller.EXISTS,
			Replicas:   int(*replicas),
			Partitions: int(*partitions),
			Conditions: []metav1.Condition{},
		},
	}
}

func (f *fixture) newController() (*Controller, informers.SharedInformerFactory, kubeinformers.SharedInformerFactory) {

	f.client = fake.NewSimpleClientset(f.objects...)
	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

	i := informers.NewSharedInformerFactory(f.client, noResyncPeriodFunc())
	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())

	c := NewController(
		f.kubeclient,
		f.client,
		i.Kafkaopscontroller().V1alpha1().KafkaTopics(),
		f.kafkaSdk)

	c.informerSynced = alwaysReady
	c.recorder = &record.FakeRecorder{}

	for _, f := range f.kafkaTopicsLister {
		err := i.Kafkaopscontroller().V1alpha1().KafkaTopics().Informer().GetIndexer().Add(f)
		if err != nil {
			return nil, nil, nil
		}
	}

	return c, i, k8sI
}

func (f *fixture) run(fooName string) {
	f.runController(fooName, true, false)
}

func (f *fixture) runExpectError(fooName string) {
	f.runController(fooName, true, true)
}

func (f *fixture) runController(fooName string, startInformers bool, expectError bool) {
	c, i, k8sI := f.newController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		i.Start(stopCh)
		k8sI.Start(stopCh)
	}

	err := c.syncHandler(fooName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing foo: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing foo, got nil")
	}

	actions := filterInformerActions(f.client.Actions())
	for i, action := range actions {
		if len(f.actions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
			break
		}

		expectedAction := f.actions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.actions) > len(actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(actions), f.actions[len(actions):])
	}

	k8sActions := filterInformerActions(f.kubeclient.Actions())
	for i, action := range k8sActions {
		if len(f.kubeactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(k8sActions)-len(f.kubeactions), k8sActions[i:])
			break
		}

		expectedAction := f.kubeactions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.kubeactions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.kubeactions)-len(k8sActions), f.kubeactions[len(k8sActions):])
	}
}

// checkAction verifies that expected and actual actions are equal and both have
// same attached resources
func checkAction(expected, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.CreateActionImpl:
		e, _ := expected.(core.CreateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.UpdateActionImpl:
		e, _ := expected.(core.UpdateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.PatchActionImpl:
		e, _ := expected.(core.PatchActionImpl)
		expPatch := e.GetPatch()
		patch := a.GetPatch()

		if !reflect.DeepEqual(expPatch, patch) {
			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expPatch, patch))
		}
	default:
		t.Errorf("Uncaptured Action %s %s, you should explicitly add a case to capture it",
			actual.GetVerb(), actual.GetResource().Resource)
	}
}

// filterInformerActions filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// nose level in our tests.
func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "kafkatopics") ||
				action.Matches("watch", "kafkatopics")) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}

func (f *fixture) expectCreateDeploymentAction(d *apps.Deployment) {
	f.kubeactions = append(f.kubeactions, core.NewCreateAction(schema.GroupVersionResource{Resource: "deployments"}, d.Namespace, d))
}

func (f *fixture) expectUpdateDeploymentAction(d *apps.Deployment) {
	f.kubeactions = append(f.kubeactions, core.NewUpdateAction(schema.GroupVersionResource{Resource: "deployments"}, d.Namespace, d))
}

func (f *fixture) expectUpdateKafkaTopicAction(foo *kafkaopscontroller.KafkaTopic) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "kafkatopics"}, foo.Namespace, foo)
	// TODO: Until #38113 is merged, we can't use Subresource
	//action.Subresource = "status"
	f.actions = append(f.actions, action)
}

func getKey(foo *kafkaopscontroller.KafkaTopic, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(foo)
	if err != nil {
		t.Errorf("Unexpected error getting key for foo %v: %v", foo.Name, err)
		return ""
	}
	return key
}

type fakeKafkaSdk struct {
	topics []string
}

func (_ fakeKafkaSdk) CreateKafkaTopic(spec kafkaopscontroller.KafkaTopicSpec) (*kafkaopscontroller.KafkaTopicStatus, error) {
	return &kafkaopscontroller.KafkaTopicStatus{
		StatusCode: kafkaopscontroller.EXISTS,
		Replicas:   1,
		Partitions: 3,
		Conditions: nil,
	}, nil
}

func (_ fakeKafkaSdk) CheckKafkaTopicStatus(spec *kafkaopscontroller.KafkaTopicSpec) (*kafkaopscontroller.KafkaTopicStatus, error) {
	return &kafkaopscontroller.KafkaTopicStatus{
		StatusCode: kafkaopscontroller.EXISTS,
		Replicas:   1,
		Partitions: 3,
		Conditions: nil,
	}, nil
}

var _ kafkaops.Interface = fakeKafkaSdk{}

func TestCreatesDeployment(t *testing.T) {
	f := newFixture(t)
	foo := newFoo("test", int32Ptr(1), int32Ptr(3))

	f.kafkaTopicsLister = append(f.kafkaTopicsLister, foo)
	f.objects = append(f.objects, foo)

	//f.expectCreateDeploymentAction(expDeployment)
	f.expectUpdateKafkaTopicAction(foo)

	f.run(getKey(foo, t))
}

func TestDoNothing(t *testing.T) {
	f := newFixture(t)
	foo := newFoo("test", int32Ptr(1), int32Ptr(3))
	//d := newKafkaTopic(foo)

	f.kafkaTopicsLister = append(f.kafkaTopicsLister, foo)
	f.objects = append(f.objects, foo)
	//f.deploymentLister = append(f.kafkaTopicsLister, foo)
	//f.kubeobjects = append(f.kubeobjects, foo)

	f.expectUpdateKafkaTopicAction(foo)
	f.run(getKey(foo, t))
}

//func TestUpdateDeployment(t *testing.T) {
//	f := newFixture(t)
//	foo := newFoo("test", int32Ptr(1))
//	d := newKafkaTopic(foo)
//
//	// Update replicas
//	foo.Spec.Replicas = int32Ptr(2)
//	expDeployment := newKafkaTopic(foo)
//
//	f.kafkaTopicsLister = append(f.kafkaTopicsLister, foo)
//	f.objects = append(f.objects, foo)
//	f.deploymentLister = append(f.deploymentLister, d)
//	f.kubeobjects = append(f.kubeobjects, d)
//
//	f.expectUpdateKafkaTopicAction(foo)
//	f.expectUpdateDeploymentAction(expDeployment)
//	f.run(getKey(foo, t))
//}
//
//func TestNotControlledByUs(t *testing.T) {
//	f := newFixture(t)
//	foo := newFoo("test", int32Ptr(1))
//	d := newKafkaTopic(foo)
//
//	d.ObjectMeta.OwnerReferences = []metav1.OwnerReference{}
//
//	f.kafkaTopicsLister = append(f.kafkaTopicsLister, foo)
//	f.objects = append(f.objects, foo)
//	f.deploymentLister = append(f.deploymentLister, d)
//	f.kubeobjects = append(f.kubeobjects, d)
//
//	f.runExpectError(getKey(foo, t))
//}

func int32Ptr(i int32) *int32 { return &i }
