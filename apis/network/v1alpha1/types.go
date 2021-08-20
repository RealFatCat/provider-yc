/*
Copyright 2020 The Crossplane Authors.

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

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// NetworkParameters are the configurable fields of a Network.
type NetworkParameters struct {
	FolderID string `json:"folder_id"`
	Name     string `json:"name"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// +optional
	Description string `json:"description,omitempty"`
}

// NetworkObservation are the observable fields of a Network.
type NetworkObservation struct {
	ID                     string            `json:"ID"`
	FolderID               string            `json:"folder_id"`
	CreatedAt              string            `json:"created_at"`
	Name                   string            `json:"name"`
	Labels                 map[string]string `json:"labels,omitempty"`
	Description            string            `json:"description,omitempty"`
	DefaultSecurityGroupID string            `json:"default_security_group_id,omitempty"`
}

// A NetworkSpec defines the desired state of a Network.
type NetworkSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       NetworkParameters `json:"forProvider"`
}

// A NetworkStatus represents the observed state of a Network.
type NetworkStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          NetworkObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Network is an API type for YC provider
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.ID"
// +kubebuilder:printcolumn:name="YC_NAME",type="string",JSONPath=".status.atProvider.name"
// +kubebuilder:printcolumn:name="FOLDER_ID",type="string",JSONPath=".status.atProvider.folder_id"
// +kubebuilder:printcolumn:name="CREATED_AT",type="string",JSONPath=".status.atProvider.created_at"
// +kubebuilder:printcolumn:name="LABELS",type="string",JSONPath=".status.atProvider.labels"
// +kubebuilder:printcolumn:name="DESCRIPTION",type="string",JSONPath=".status.atProvider.description"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,yc}
type Network struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkSpec   `json:"spec"`
	Status NetworkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkList contains a list of Network
type NetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Network `json:"items"`
}
