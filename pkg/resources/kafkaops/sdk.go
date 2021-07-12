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
	TopicStatus v1alpha1.TopicStatusCode
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

func CreateFooTopic(spec v1alpha1.FooSpec) (*v1alpha1.TopicStatus, error) {
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

	return &v1alpha1.TopicStatus{
		StatusCode: v1alpha1.UNKNOWN,
		Replicas:   0,
		Partitions: 0,
	}, nil
}

func GetTopicStatus(spec *v1alpha1.FooSpec) (*v1alpha1.TopicStatus, error) {

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
		[]kafka.ConfigResource{{Type: kafka.ResourceTopic, Name: spec.TopicName}},
		kafka.SetAdminRequestTimeout(dur))
	if err != nil {
		fmt.Printf("Failed to DescribeConfigs(%s, %s): %s\n",
			kafka.ResourceTopic, spec.TopicName, err)
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
	status := v1alpha1.UNKNOWN
	if result.Error.Code() == kafka.ErrNoError {
		status = v1alpha1.EXISTS
	} else if result.Error.Code() == kafka.ErrUnknownTopicOrPart {
		status = v1alpha1.NOT_EXISTS
	}

	return &v1alpha1.TopicStatus{
		StatusCode: status,
		Replicas:   1,
		Partitions: 1,
	}, nil
}
