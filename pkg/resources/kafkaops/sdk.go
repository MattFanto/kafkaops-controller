package kafkaops

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/mattfanto/kafkaops-controller/pkg/apis/kafkaopscontroller/v1alpha1"
	"time"
)

// Interface for the controller to interact with the Kafka cluster
// the idea is that this package implements and use model from k8s (e.g. KafkaTopic CRD as input)
type Interface interface {
	CreateKafkaTopic(spec *v1alpha1.KafkaTopic) (*v1alpha1.KafkaTopicStatus, error)
	CheckKafkaTopicStatus(spec *v1alpha1.KafkaTopic) (*v1alpha1.KafkaTopicStatus, error)
}

type KafkaSdk struct {
	adminClient *kafka.AdminClient
}

func NewKafkaSdk(bootstrapServer string) (Interface, error) {
	cf := kafka.ConfigMap{
		"bootstrap.servers": bootstrapServer,
	}
	a, err := kafka.NewAdminClient(&cf)
	if err != nil {
		return nil, err
	}
	kafkaSdk := &KafkaSdk{
		adminClient: a,
	}
	return kafkaSdk, nil
}

// CreateKafkaTopic creates a topic in Kafka according to the specification defined in KafkaTopicSpec
func (kafkaSdk *KafkaSdk) CreateKafkaTopic(kafkaTopic *v1alpha1.KafkaTopic) (*v1alpha1.KafkaTopicStatus, error) {
	spec := kafkaTopic.Spec
	// Contexts are used to abort or limit the amount of time
	// the Admin call blocks waiting for a result.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results, err := kafkaSdk.adminClient.CreateTopics(
		ctx,
		// Multiple topics can be created simultaneously
		// by providing more TopicSpecification structs here.
		[]kafka.TopicSpecification{{
			Topic:             spec.TopicName,
			NumPartitions:     int(*spec.Replicas),
			ReplicationFactor: int(*spec.Replicas)}},
		// Admin options
		kafka.SetAdminOperationTimeout(60*time.Second))
	if err != nil {
		fmt.Printf("Failed to create topic: %v\n", err)
		return nil, err
	}
	if len(results) != 1 {
		panic("Expected one results after issuing create for one topic")
	}
	result := results[0]

	/**
	TODO remap all errors
	*/
	if result.Error.Code() == kafka.ErrTopicAlreadyExists {

	}

	return &v1alpha1.KafkaTopicStatus{
		StatusCode: v1alpha1.UNKNOWN,
		Replicas:   0,
		Partitions: 0,
	}, nil
}

// CheckKafkaTopicStatus performs several checks on a Kafka topic according to the specification
// defined in KafkaTopicSpec.
// In particular the following checks are performed:
// * topic existence
// * topic configuration matches
func (kafkaSdk *KafkaSdk) CheckKafkaTopicStatus(kafkaTopic *v1alpha1.KafkaTopic) (*v1alpha1.KafkaTopicStatus, error) {
	spec := kafkaTopic.Spec
	// Contexts are used to abort or limit the amount of time
	// the Admin call blocks waiting for a config.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Results will be stored here
	topicStatus := v1alpha1.KafkaTopicStatus{}

	dur, _ := time.ParseDuration("20s")
	configs, err := kafkaSdk.adminClient.DescribeConfigs(ctx,
		[]kafka.ConfigResource{{Type: kafka.ResourceTopic, Name: spec.TopicName}},
		kafka.SetAdminRequestTimeout(dur))
	if err != nil {
		fmt.Printf("Failed to DescribeConfigs(%s, %s): %s\n",
			kafka.ResourceTopic, spec.TopicName, err)
		return nil, err
	}
	if len(configs) != 1 {
		panic("Expected one configs after issuing create for one topic")
	}
	config := configs[0]

	/**
	TODO remap errors
	*/
	switch config.Error.Code() {
	case kafka.ErrNoError:
		topicStatus.StatusCode = v1alpha1.EXISTS
	case kafka.ErrUnknownTopicOrPart:
		topicStatus.StatusCode = v1alpha1.NOT_EXISTS
	default:
		topicStatus.StatusCode = v1alpha1.UNKNOWN
	}

	metadata, err := kafkaSdk.adminClient.GetMetadata(&spec.TopicName, false, int(dur.Milliseconds()))
	if err != nil {
		return nil, err
	}
	partitions := metadata.Topics[spec.TopicName].Partitions
	if len(partitions) > 0 {
		topicStatus.Partitions = len(partitions)
		topicStatus.Replicas = len(partitions[0].Replicas)
	}

	return &topicStatus, nil
}
