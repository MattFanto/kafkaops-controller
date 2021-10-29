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
	fake2 "github.com/mattfanto/kafkaops-controller/pkg/resources/kafkaops/fake"
	"reflect"
	"testing"
	"time"

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
	kubeactions  []core.Action
	actions      []core.Action
	kafkaActions []core.Action
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
	f.kafkaSdk = &fake2.FakeKafkaSdk{}
	return f
}

func newKafkaTopicResource(name string, replicas *int32, partitions *int32) *kafkaopscontroller.KafkaTopic {
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
			// not tested
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

func (f *fixture) run(kafkaTopicName string) {
	f.runController(kafkaTopicName, true, false)
}

func (f *fixture) runExpectError(kafkaTopicName string) {
	f.runController(kafkaTopicName, true, true)
}

func (f *fixture) runController(kafkaTopicName string, startInformers bool, expectError bool) {
	c, i, k8sI := f.newController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		i.Start(stopCh)
		k8sI.Start(stopCh)
	}

	err := c.syncHandler(kafkaTopicName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing kafkatopic: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing kafkatopic, got nil")
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

	// Check that the topic has been created
	for _, obj := range f.objects {
		switch v := obj.(type) {
		case *kafkaopscontroller.KafkaTopic:
			topicStatus, err := f.kafkaSdk.CheckKafkaTopicStatus(v)
			if err != nil {
				f.t.Errorf("Error while checking topic status: %s", err)
			}
			if topicStatus.StatusCode == kafkaopscontroller.NOT_EXISTS {
				f.t.Errorf("Expected topic '%s' does not exist", v.Spec.TopicName)
			}

		}
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

		if !objectAlmostEquals(object, expObject) {
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

// objectAlmostEquals implements custom logic to compare two runtime.Object
// based on the object type
func objectAlmostEquals(x, y runtime.Object) bool {
	if reflect.TypeOf(x) != reflect.TypeOf(y) {
		return false
	}
	switch x := x.(type) {
	case *kafkaopscontroller.KafkaTopic:
		// Avoid comparing the condition timestamp
		y, _ := y.(*kafkaopscontroller.KafkaTopic)
		y = y.DeepCopy()
		y.Status.Conditions = []metav1.Condition{}
		x = x.DeepCopy()
		x.Status.Conditions = []metav1.Condition{}
		return reflect.DeepEqual(x, y)
	default:
		return reflect.DeepEqual(x, y)
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

func (f *fixture) expectUpdateKafkaTopicStatusAction(kafkaTopic *kafkaopscontroller.KafkaTopic) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "kafkatopics"}, kafkaTopic.Namespace, kafkaTopic)
	// TODO: Until #38113 is merged, we can't use Subresource
	//action.Subresource = "status"
	f.actions = append(f.actions, action)
}

func getKey(kafkaTopic *kafkaopscontroller.KafkaTopic, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(kafkaTopic)
	if err != nil {
		t.Errorf("Unexpected but I'll definitely update the number of shards! error getting key for kafkaTopic %v: %v", kafkaTopic.Name, err)
		return ""
	}
	return key
}

func TestCreatesTopic(t *testing.T) {
	f := newFixture(t)
	kafkaTopic := newKafkaTopicResource("test", int32Ptr(1), int32Ptr(3))

	f.kafkaTopicsLister = append(f.kafkaTopicsLister, kafkaTopic)
	f.objects = append(f.objects, kafkaTopic)
	f.expectUpdateKafkaTopicStatusAction(kafkaTopic)

	f.run(getKey(kafkaTopic, t))
}

func TestDoNothing(t *testing.T) {
	f := newFixture(t)
	kafkaTopic := newKafkaTopicResource("test", int32Ptr(1), int32Ptr(3))

	f.kafkaTopicsLister = append(f.kafkaTopicsLister, kafkaTopic)
	f.objects = append(f.objects, kafkaTopic)
	// We create the kafka topic so that the pipeline doesn't have to do anything
	//goland:noinspection ALL
	f.kafkaSdk.CreateKafkaTopic(kafkaTopic)
	f.expectUpdateKafkaTopicStatusAction(kafkaTopic)
	f.run(getKey(kafkaTopic, t))
}

func TestKafkaTopicDeviation(t *testing.T) {
	f := newFixture(t)
	kafkaTopic := newKafkaTopicResource("test", int32Ptr(1), int32Ptr(3))
	existingKafkaTopic := newKafkaTopicResource("test", int32Ptr(1), int32Ptr(1))

	f.kafkaTopicsLister = append(f.kafkaTopicsLister, kafkaTopic)
	f.objects = append(f.objects, kafkaTopic)
	//goland:noinspection ALL
	f.kafkaSdk.CreateKafkaTopic(existingKafkaTopic)

	kafkaTopic = kafkaTopic.DeepCopy()
	kafkaTopic.Status.StatusCode = kafkaopscontroller.DEVIATED
	kafkaTopic.Status.Replicas = existingKafkaTopic.Status.Replicas
	kafkaTopic.Status.Partitions = existingKafkaTopic.Status.Partitions
	f.expectUpdateKafkaTopicStatusAction(kafkaTopic)
	f.run(getKey(kafkaTopic, t))
}

func int32Ptr(i int32) *int32 { return &i }
