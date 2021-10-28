package fake

import (
	"fmt"
	kafkaopscontroller "github.com/mattfanto/kafkaops-controller/pkg/apis/kafkaopscontroller/v1alpha1"
	"github.com/mattfanto/kafkaops-controller/pkg/resources/kafkaops"
	"k8s.io/apimachinery/pkg/runtime/schema"
	core "k8s.io/client-go/testing"
)

// FakeKafkaSdk implement sdk.Interface mocking a kafka broker
// created topic and existing topic are stored in topics array
// while kafkaActions store a reference to the API call to the
// interface to check which operation are applied to the broker
// at the end of the test session
type FakeKafkaSdk struct {
	kafkaActions []core.Action
	topics       []*kafkaopscontroller.KafkaTopic
}

func (this *FakeKafkaSdk) getExistingTopic(topic *kafkaopscontroller.KafkaTopic) *kafkaopscontroller.KafkaTopic {
	for _, val := range this.topics {
		if val.Spec.TopicName == topic.Spec.TopicName {
			return val
		}
	}
	return nil
}

func (this *FakeKafkaSdk) CreateKafkaTopic(kafkaTopic *kafkaopscontroller.KafkaTopic) (*kafkaopscontroller.KafkaTopicStatus, error) {
	this.kafkaActions = append(
		this.kafkaActions,
		core.NewCreateAction(
			schema.GroupVersionResource{Resource: "kafkatopics"},
			kafkaTopic.Namespace,
			kafkaTopic,
		),
	)
	if this.getExistingTopic(kafkaTopic) != nil {
		return nil, fmt.Errorf("topic already exists")
	}
	this.topics = append(this.topics, kafkaTopic)
	return &kafkaopscontroller.KafkaTopicStatus{
		StatusCode: kafkaopscontroller.EXISTS,
		Replicas:   1,
		Partitions: 3,
		Conditions: nil,
	}, nil
}

func (this *FakeKafkaSdk) CheckKafkaTopicStatus(kafkaTopic *kafkaopscontroller.KafkaTopic) (*kafkaopscontroller.KafkaTopicStatus, error) {
	existingTopic := this.getExistingTopic(kafkaTopic)
	if existingTopic != nil {
		return &kafkaopscontroller.KafkaTopicStatus{
			StatusCode: kafkaopscontroller.EXISTS,
			Replicas:   int(*existingTopic.Spec.Replicas),
			Partitions: int(*existingTopic.Spec.Partitions),
			Conditions: nil,
		}, nil
	} else {
		return &kafkaopscontroller.KafkaTopicStatus{
			StatusCode: kafkaopscontroller.NOT_EXISTS,
			Replicas:   0,
			Partitions: 0,
			Conditions: nil,
		}, nil
	}
}

var _ kafkaops.Interface = &FakeKafkaSdk{}
