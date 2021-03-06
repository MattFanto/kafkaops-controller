/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaTopic is a specification for a KafkaTopic resource
type KafkaTopic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaTopicSpec   `json:"spec"`
	Status KafkaTopicStatus `json:"status"`
}

// KafkaTopicSpec is the spec for a KafkaTopic resource
type KafkaTopicSpec struct {
	TopicName  string `json:"topicName"`
	Replicas   *int32 `json:"replicas"`
	Partitions *int32 `json:"partitions"`
}

type TopicStatusCode string

const (
	UNKNOWN    TopicStatusCode = "UNKNOWN"
	EXISTS     TopicStatusCode = "EXISTS"
	NOT_EXISTS TopicStatusCode = "NOT_EXISTS"
	DEVIATED   TopicStatusCode = "DEVIATED"
)

type KafkaTopicStatus struct {
	StatusCode TopicStatusCode    `json:"statusCode"`
	Replicas   int                `json:"replicas"`
	Partitions int                `json:"partitions"`
	Conditions []metav1.Condition `json:"conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaTopicList is a list of KafkaTopic resources
type KafkaTopicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []KafkaTopic `json:"items"`
}
