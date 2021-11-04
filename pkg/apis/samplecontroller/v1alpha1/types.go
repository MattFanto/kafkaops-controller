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

// Foo is a specification for a Foo resource
type Foo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FooSpec     `json:"spec"`
	Status TopicStatus `json:"status"`
}

// FooSpec is the spec for a Foo resource
type FooSpec struct {
	TopicName  string `json:"topicName"`
	Replicas   *int32 `json:"replicas"`
	Partitions *int32 `json:"partitions"`
}

type TopicStatusCode string

const (
	UNKNOWN    TopicStatusCode = "UNKNOWN"
	EXISTS     TopicStatusCode = "EXISTS"
	NOT_EXISTS TopicStatusCode = "NOT_EXISTS"
	// TODO deviation status
	// DEVIATED TopicStatusCode = "DEVIATED"
)

type TopicStatus struct {
	StatusCode TopicStatusCode `json:"statusCode"`
	Replicas   int             `json:"replicas"`
	Partitions int             `json:"partitions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FooList is a list of Foo resources
type FooList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Foo `json:"items"`
}
