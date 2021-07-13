package tests

import (
	"github.com/mattfanto/kafkaops-controller/pkg/apis/kafkaopscontroller/v1alpha1"
	"github.com/mattfanto/kafkaops-controller/pkg/resources/kafkaops"
	"github.com/stretchr/testify/assert"
	"testing"
)

func int32Ptr(i int32) *int32 { return &i }

func TestCreateTopic(t *testing.T) {
	_, err := kafkaops.CreateFooTopic(v1alpha1.KafkaTopicSpec{
		TopicName: "example_topic_v1",
		Replicas:  int32Ptr(1),
	})
	if err != nil {
		return
	}
}

func TestGetStatus(t *testing.T) {
	topicStatus, err := kafkaops.CheckKafkaTopicStatus(&v1alpha1.KafkaTopicSpec{
		TopicName: "example_topic_not",
		Replicas:  int32Ptr(1),
	})
	if err != nil {
		return
	}
	assert.Equal(t, "NOT_EXISTS", topicStatus.TopicStatus, "Expected topic to not exists")
}
