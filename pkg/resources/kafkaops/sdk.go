package kafkaops

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/mattfanto/kafkaops-controller/pkg/apis/kafkaopscontroller/v1alpha1"
	"time"
)

func getClient() (*kafka.AdminClient, error) {
	cf := kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	}
	a, err := kafka.NewAdminClient(&cf)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func CreateKafkaTopic(spec v1alpha1.KafkaTopicSpec) (*v1alpha1.KafkaTopicStatus, error) {
	maxDur, err := time.ParseDuration("60s")
	if err != nil {
		panic("ParseDuration(60s)")
	}
	a, err := getClient()
	if err != nil {
		return nil, err
	}
	// Contexts are used to abort or limit the amount of time
	// the Admin call blocks waiting for a result.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results, err := a.CreateTopics(
		ctx,
		// Multiple topics can be created simultaneously
		// by providing more TopicSpecification structs here.
		[]kafka.TopicSpecification{{
			Topic:             spec.TopicName,
			NumPartitions:     int(*spec.Replicas),
			ReplicationFactor: int(*spec.Replicas)}},
		// Admin options
		kafka.SetAdminOperationTimeout(maxDur))
	if err != nil {
		fmt.Printf("Failed to create topic: %v\n", err)
		return nil, err
	}
	if len(results) != 1 {
		panic("Expected one results after issuing create for one topic")
	}
	a.Close()
	result := results[0]

	/**
	TODO remap errors
	*/
	if result.Error.Code() == kafka.ErrTopicAlreadyExists {

	}

	return &v1alpha1.KafkaTopicStatus{
		StatusCode: v1alpha1.UNKNOWN,
		Replicas:   0,
		Partitions: 0,
	}, nil
}

func CheckKafkaTopicStatus(spec *v1alpha1.KafkaTopicSpec) (*v1alpha1.KafkaTopicStatus, error) {

	a, err := getClient()
	if err != nil {
		return nil, err
	}
	// Contexts are used to abort or limit the amount of time
	// the Admin call blocks waiting for a config.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Results will be stored here
	topicStatus := v1alpha1.KafkaTopicStatus{}

	dur, _ := time.ParseDuration("20s")
	configs, err := a.DescribeConfigs(ctx,
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
	if config.Error.Code() == kafka.ErrNoError {
		topicStatus.StatusCode = v1alpha1.EXISTS
	} else if config.Error.Code() == kafka.ErrUnknownTopicOrPart {
		topicStatus.StatusCode = v1alpha1.NOT_EXISTS
	} else {
		topicStatus.StatusCode = v1alpha1.UNKNOWN
	}

	metadata, err := a.GetMetadata(&spec.TopicName, false, int(dur.Milliseconds()))
	if err != nil {
		return nil, err
	}
	partitions := metadata.Topics[spec.TopicName].Partitions
	if len(partitions) > 0 {
		topicStatus.Partitions = len(partitions)
		topicStatus.Replicas = len(partitions[0].Replicas)
	}

	a.Close()

	return &topicStatus, nil
}
