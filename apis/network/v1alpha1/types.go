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

// NetworkTypeParameters are the configurable fields of a NetworkType.
type NetworkTypeParameters struct {
	Name string `json:"name"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// +optional
	Description string `json:"description,omitempty"`
}

// NetworkTypeObservation are the observable fields of a NetworkType.
type NetworkTypeObservation struct {
	ID          string            `json:"ID,omitempty"`
	FolderID    string            `json:"folder_id"`
	CreatedAt   string            `json:"created_at"`
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Description string            `json:"description,omitempty"`
}

// A NetworkTypeSpec defines the desired state of a NetworkType.
type NetworkTypeSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       NetworkTypeParameters `json:"forProvider"`
}

// A NetworkTypeStatus represents the observed state of a NetworkType.
type NetworkTypeStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          NetworkTypeObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A NetworkType is an API type for YC provider
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.bindingPhase"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,yc}
type NetworkType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkTypeSpec   `json:"spec"`
	Status NetworkTypeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkTypeList contains a list of NetworkType
type NetworkTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkType `json:"items"`
}
