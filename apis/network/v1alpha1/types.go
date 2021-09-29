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

// NetworkObservation are the observable fields of a Network.
type NetworkObservation struct {
	ID                     string            `json:"ID"`
	FolderID               string            `json:"folderId"`
	CreatedAt              string            `json:"createdAt"`
	Name                   string            `json:"name"`
	Labels                 map[string]string `json:"labels,omitempty"`
	Description            string            `json:"description,omitempty"`
	DefaultSecurityGroupID string            `json:"defaultSecurityGroupId,omitempty"`
}

// A NetworkSpec defines the desired state of a Network.
type NetworkSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	FolderID          string `json:"folderId"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// +optional
	Description string `json:"description,omitempty"`
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

// DhcpOptions contains dhcp options for Subnet
type DhcpOptions struct {
	DomainNameServers []string `json:"domainNameServers,omitempty"`
	DomainName        string   `json:"domainName,omitempty"`
	NtpServers        []string `json:"ntpServers,omitempty"`
}

// SubnetObservation are the observable fields of a Subnet.
type SubnetObservation struct {
	ID           string            `json:"ID"`
	Name         string            `json:"name"`
	NetworkID    string            `json:"networkId"`
	NetworkName  string            `json:"networkName"`
	ZoneID       string            `json:"zoneId"`
	Range        string            `json:"range"`
	Description  string            `json:"description,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	CreatedAt    string            `json:"createdAt"`
	FolderID     string            `json:"folderId"`
	RouteTableID string            `json:"routeTableId,omitempty"`
	V4CidrBlocks []string          `json:"v4CidrBlocks,omitempty"`
	V6CidrBlocks []string          `json:"v6CidrBlocks,omitempty"` // IPv6 not available yet.
	DhcpOptions  string            `json:"dhcpOptions,omitempty"`  // not supported by us yet.
}

// A SubnetSpec defines the desired state of a Subnet.
type SubnetSpec struct {
	xpv1.ResourceSpec `json:",inline"`

	FolderID string `json:"folderId"`
	ZoneID   string `json:"zoneId"`
	// +optional
	NetworkID string `json:"networkId,omitempty"`
	// +optional
	NetworkIDRef *xpv1.Reference `json:"networkIdRef,omitempty"`
	// +optional
	NetworkIDSelector *xpv1.Selector `json:"networkIdSelector,omitempty"`
	V4CidrBlocks      []string       `json:"v4CidrBlocks"`
	// +optional
	Description string `json:"description,omitempty"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// +optional
	RouteTableID string `json:"routeTableId,omitempty"`
	// +optional
	DhcpOptions *DhcpOptions `json:"dhcpOptions,omitempty"`
}

// A SubnetStatus represents the observed state of a Subnet.
type SubnetStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          SubnetObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Subnet is an API type for YC provider
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="NETWORK_NAME",type="string",JSONPath=".status.atProvider.networkName"
// +kubebuilder:printcolumn:name="RANGE",type="string",JSONPath=".status.atProvider.range"
// +kubebuilder:printcolumn:name="ZONE_ID",type="string",JSONPath=".status.atProvider.zoneId"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,yc}
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubnetSpec   `json:"spec"`
	Status SubnetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SubnetList contains a list of Subnet
type SubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Subnet `json:"items"`
}
