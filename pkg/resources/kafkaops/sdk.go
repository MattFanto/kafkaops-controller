package kafkaops

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"k8s.io/sample-controller/pkg/apis/samplecontroller/v1alpha1"
	"time"
)

type KafkaTopicStatus struct {
	TopicName   string
	TopicStatus string
}

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

func CreateFooTopic(spec v1alpha1.FooSpec) (*KafkaTopicStatus, error) {
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
			Topic:             spec.DeploymentName,
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

	return &KafkaTopicStatus{
		TopicName:   result.Topic,
		TopicStatus: "UNCHECKED",
	}, nil
}

func GetTopicStatus(spec *v1alpha1.FooSpec) (*KafkaTopicStatus, error) {

	a, err := getClient()
	if err != nil {
		return nil, err
	}
	// Contexts are used to abort or limit the amount of time
	// the Admin call blocks waiting for a result.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dur, _ := time.ParseDuration("20s")
	results, err := a.DescribeConfigs(ctx,
		[]kafka.ConfigResource{{Type: kafka.ResourceTopic, Name: spec.DeploymentName}},
		kafka.SetAdminRequestTimeout(dur))
	if err != nil {
		fmt.Printf("Failed to DescribeConfigs(%s, %s): %s\n",
			kafka.ResourceTopic, spec.DeploymentName, err)
		return nil, err
	}
	if len(results) != 1 {
		panic("Expected one results after issuing create for one topic")
	}
	result := results[0]

	a.Close()
	/**
	TODO remap errors
	*/
	status := "UNCHECKED"
	if result.Error.Code() == kafka.ErrNoError {
		status = "EXISTS"
	} else if result.Error.Code() == kafka.ErrUnknownTopicOrPart {
		status = "NOT_EXISTS"
	}

	return &KafkaTopicStatus{
		TopicName:   spec.DeploymentName,
		TopicStatus: status,
	}, nil
}
