/**
These tests won't work without Kafka up and running I need to create a fake for kafka-go-library, but this is out
of scope for the moment
*/
package kafkaops

import (
	kafkaopscontroller "github.com/mattfanto/kafkaops-controller/pkg/apis/kafkaopscontroller/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math/rand"
	"testing"
	"time"
)

func int32Ptr(i int32) *int32 { return &i }

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randTopicName() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return "test_topic_" + string(b)
}

func getKafkaSdk(t *testing.T) Interface {
	kafkaSdk, err := NewKafkaSdk("localhost:9092")
	if err != nil {
		t.Errorf("Unable to create kafkasdr, error: %s", err)
	}
	return kafkaSdk
}

func newKafkaTopicResource(name string, replicas int, partitions int) *kafkaopscontroller.KafkaTopic {
	return &kafkaopscontroller.KafkaTopic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kafkaopscontroller.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: kafkaopscontroller.KafkaTopicSpec{
			TopicName:  name,
			Replicas:   int32Ptr(int32(replicas)),
			Partitions: int32Ptr(int32(partitions)),
		},
		Status: kafkaopscontroller.KafkaTopicStatus{
			StatusCode: kafkaopscontroller.EXISTS,
			Replicas:   replicas,
			Partitions: partitions,
			// not tested
			Conditions: []metav1.Condition{},
		},
	}
}

func TestCreateTopicAndGetStatus(t *testing.T) {
	kafkaSdk := getKafkaSdk(t)
	kafkaTopic := newKafkaTopicResource(randTopicName(), 1, 1)
	_, err := kafkaSdk.CreateKafkaTopic(kafkaTopic)
	if err != nil {
		t.Errorf("Unable to create kafkatopic, error: %s", err)
	}
	topicStatus, err := kafkaSdk.CheckKafkaTopicStatus(kafkaTopic)
	if err != nil {
		t.Errorf("Error while retrieving topic status, error: %s", err)
	}
	assert.Equal(t, kafkaopscontroller.EXISTS, topicStatus.StatusCode, "Expected topic to not exists")
}

func TestGetStatus(t *testing.T) {
	kafkaSdk := getKafkaSdk(t)
	kafkaTopic := newKafkaTopicResource(randTopicName(), 1, 1)
	topicStatus, err := kafkaSdk.CheckKafkaTopicStatus(kafkaTopic)
	if err != nil {
		t.Errorf("Error while retrieving topic status, error: %s", err)
	}
	assert.Equal(t, kafkaopscontroller.NOT_EXISTS, topicStatus.StatusCode, "Expected topic to not exists")
}
