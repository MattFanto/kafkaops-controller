package tests

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/sample-controller/pkg/apis/samplecontroller/v1alpha1"
	"k8s.io/sample-controller/pkg/resources/kafkaops"
	"testing"
)

func int32Ptr(i int32) *int32 { return &i }

func TestCreateTopic(t *testing.T) {
	_, err := kafkaops.CreateFooTopic(v1alpha1.FooSpec{
		DeploymentName: "example_topic_v1",
		Replicas:       int32Ptr(1),
	})
	if err != nil {
		return
	}
}

func TestGetStatus(t *testing.T) {
	topicStatus, err := kafkaops.GetTopicStatus(&v1alpha1.FooSpec{
		DeploymentName: "example_topic_not",
		Replicas:       int32Ptr(1),
	})
	if err != nil {
		return
	}
	assert.Equal(t, "NOT_EXISTS", topicStatus.TopicStatus, "Expected topic to not exists")
}
