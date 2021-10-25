package main

import (
	"context"
	"fmt"
	"github.com/mattfanto/kafkaops-controller/pkg/resources/kafkaops"
	"k8s.io/apimachinery/pkg/api/meta"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	samplev1alpha1 "github.com/mattfanto/kafkaops-controller/pkg/apis/kafkaopscontroller/v1alpha1"
	clientset "github.com/mattfanto/kafkaops-controller/pkg/generated/clientset/versioned"
	samplescheme "github.com/mattfanto/kafkaops-controller/pkg/generated/clientset/versioned/scheme"
	informers "github.com/mattfanto/kafkaops-controller/pkg/generated/informers/externalversions/kafkaopscontroller/v1alpha1"
	listers "github.com/mattfanto/kafkaops-controller/pkg/generated/listers/kafkaopscontroller/v1alpha1"
)

const controllerAgentName = "kafkaops-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a KafkaTopic is synced
	SuccessSynced = "Synced"
	// ErrResourceDeviated is used as part of the Event 'reason' when a KafkaTopic deviates
	// from the original specification
	ErrResourceDeviated = "ErrResourceDeviated"

	// MessageResourceSynced is the message used for an Event fired when a KafkaTopic
	// is synced successfully
	MessageResourceSynced = "KafkaTopic synced successfully"
	// MessageResourceDeviated is used as part of the Event 'reason' when a KafkaTopic deviates from the
	// original specification
	MessageResourceDeviated = "KafkaTopic configuration deviated from the configuration"
)

// Controller is the controller implementation for KafkaTopic resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	sampleclientset clientset.Interface

	kafkaTopicsLister listers.KafkaTopicLister
	informerSynced    cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
	//
	kafkaSdk kafkaops.Interface
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	sampleclientset clientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	fooInformer informers.KafkaTopicInformer,
	kafkaSdk kafkaops.Interface,
) *Controller {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(samplescheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:     kubeclientset,
		sampleclientset:   sampleclientset,
		kafkaTopicsLister: fooInformer.Lister(),
		informerSynced:    fooInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Foos"),
		recorder:          recorder,
		kafkaSdk:          kafkaSdk,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when KafkaTopic resources change
	fooInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueFoo,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueFoo(new)
		},
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a KafkaTopic resource then the handler will enqueue that KafkaTopic resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.Deployment)
			oldDepl := old.(*appsv1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting KafkaTopic controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.informerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process KafkaTopic resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// KafkaTopic resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
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

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the KafkaTopic resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the KafkaTopic resource with this namespace/name
	kafkaTopic, err := c.kafkaTopicsLister.KafkaTopics(namespace).Get(name)
	if err != nil {
		// The KafkaTopic resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("kafkaTopic '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	deploymentName := kafkaTopic.Spec.TopicName
	if deploymentName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s: topicStatus name must be specified", key))
		return nil
	}

	// Get the topicStatus with the name specified in KafkaTopic.spec
	topicStatus, err := checkKafkaTopic(c, kafkaTopic)
	if err != nil {
		return err
	}
	// If the resource doesn't exist, we'll create it
	if topicStatus.StatusCode == samplev1alpha1.NOT_EXISTS {
		topicStatus, err = newKafkaTopic(c, kafkaTopic)
	}
	klog.Info(topicStatus)

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	/**
	TODO nice to have but need some topic metadata
	// If the topic is not controlled by this KafkaTopic resource, we should log
	// a warning to the event recorder and return error msg.
	if !metav1.IsControlledBy(topicStatus, kafkaTopic) {
		msg := fmt.Sprintf(MessageResourceExists, topicStatus.Name)
		c.recorder.Event(kafkaTopic, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}
	*/

	/**
	TODO nice to have regarding topic config

	// If this number of the replicas on the KafkaTopic resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the topic is specified in settings.
	if kafkaTopic.Spec.Replicas != nil && *kafkaTopic.Spec.Replicas != *topicStatus.Spec.Replicas {
		klog.V(4).Infof("KafkaTopic %s replicas: %d, topicStatus replicas: %d", name, *kafkaTopic.Spec.Replicas, *topicStatus.Spec.Replicas)
		topicStatus, err = c.kubeclientset.AppsV1().Deployments(kafkaTopic.Namespace).Update(context.TODO(), newKafkaTopic(kafkaTopic), metav1.UpdateOptions{})
	}
	*/

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// Finally, we update the status block of the KafkaTopic resource to reflect the
	// current state of the world
	err = c.updateKafkaTopicStatus(kafkaTopic, topicStatus)
	if err != nil {
		return err
	}

	if topicStatus.StatusCode == samplev1alpha1.DEVIATED {
		c.recorder.Event(kafkaTopic, corev1.EventTypeWarning, ErrResourceDeviated, MessageResourceDeviated)
	} else {
		c.recorder.Event(kafkaTopic, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	}
	return nil
}

func (c *Controller) updateKafkaTopicStatus(kafkaTopic *samplev1alpha1.KafkaTopic, topicStatus *samplev1alpha1.KafkaTopicStatus) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	kafkaTopicCopy := kafkaTopic.DeepCopy()
	kafkaTopicCopy.Status = *topicStatus
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the KafkaTopic resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.sampleclientset.KafkaopscontrollerV1alpha1().KafkaTopics(kafkaTopic.Namespace).Update(context.TODO(), kafkaTopicCopy, metav1.UpdateOptions{})
	return err
}

// enqueueFoo takes a KafkaTopic resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than KafkaTopic.
func (c *Controller) enqueueFoo(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the KafkaTopic resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that KafkaTopic resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a KafkaTopic, we should not do anything more
		// with it.
		if ownerRef.Kind != "KafkaTopic" {
			return
		}

		foo, err := c.kafkaTopicsLister.KafkaTopics(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of foo '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueFoo(foo)
		return
	}
}

// newKafkaTopic creates a new topic in kafka for a KafkaTopic resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the KafkaTopic resource that 'owns' it.
//
// This will now create a new topic
func newKafkaTopic(c *Controller, kafkaTopic *samplev1alpha1.KafkaTopic) (*samplev1alpha1.KafkaTopicStatus, error) {
	topic, err := c.kafkaSdk.CreateKafkaTopic(kafkaTopic.Spec)
	if err != nil {
		return nil, err
	}
	klog.Infof("Created topic named '%s'", kafkaTopic.Spec.TopicName)
	return topic, nil
}

// checkKafkaTopic compare the topic configuration against specification in KafkaTopic resource.
// KafkaTopic resource status will be updated accordingly.
func checkKafkaTopic(c *Controller, kafkaTopic *samplev1alpha1.KafkaTopic) (*samplev1alpha1.KafkaTopicStatus, error) {
	topicStatus, err := c.kafkaSdk.CheckKafkaTopicStatus(&kafkaTopic.Spec)
	if err != nil {
		return nil, err
	}
	klog.Infof("Topic status '%s'", topicStatus.StatusCode)
	if topicStatus.StatusCode == samplev1alpha1.EXISTS {
		// Since the topic exists we can check possible deviation
		if kafkaTopic.Spec.Replicas != nil && *kafkaTopic.Spec.Replicas != int32(topicStatus.Replicas) {
			meta.SetStatusCondition(&topicStatus.Conditions, metav1.Condition{
				Type:    "Ready",
				Status:  metav1.ConditionFalse,
				Message: "Replicas count mismatch between status and specification",
			})
			topicStatus.StatusCode = samplev1alpha1.DEVIATED
		} else if kafkaTopic.Spec.Partitions != nil && *kafkaTopic.Spec.Partitions != int32(topicStatus.Partitions) {
			meta.SetStatusCondition(&topicStatus.Conditions, metav1.Condition{
				Type:    "Ready",
				Status:  metav1.ConditionFalse,
				Message: "Replicas count mismatch between status and specification",
			})
			topicStatus.StatusCode = samplev1alpha1.DEVIATED
		} else {
			meta.SetStatusCondition(&topicStatus.Conditions, metav1.Condition{
				Type:    "Ready",
				Status:  metav1.ConditionTrue,
				Message: "Topic ready and specification in sync",
			})
		}
	}
	return topicStatus, nil
}
