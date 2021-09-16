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

type InternalAddressSpec struct {
	// ID of the subnet. If no ID is specified, and there only one subnet in specified zone, an address in this subnet will be allocated.
	// +optional
	SubnetId string `json:"subnet_id,omitempty"`
}

type ExternalAddressSpec struct{}

type ZonalMasterSpec struct {
	// ID of the availability zone.
	// +optional
	ZoneId string `json:"zone_id,omitempty"`
	// Specification of parameters for internal IPv4 networking.
	// +optional
	InternalV4AddressSpec *InternalAddressSpec `json:"internal_v4_address_spec,omitempty"`
	// Specification of parameters for external IPv4 networking.
	// +optional
	ExternalV4AddressSpec *ExternalAddressSpec `json:"external_v4_address_spec,omitempty"`
}

type RegionalMasterSpec struct {
	// ID of the availability zone where the master resides.
	// +optional
	RegionId string `json:"region_id,omitempty"`
	// List of locations where the master will be allocated.
	// +optional
	Locations []*MasterLocation `json:"locations,omitempty"`
	// Specify to allocate a static public IP for the master.
	// +optional
	ExternalV4AddressSpec *ExternalAddressSpec `json:"external_v4_address_spec,omitempty"`
}

type MasterLocation struct {
	// ID of the availability zone.
	// +optional
	ZoneId string `json:"zone_id,omitempty"`
	// If not specified and there is a single subnet in specified zone, address
	// in this subnet will be allocated.
	// +optional
	InternalV4AddressSpec *InternalAddressSpec `json:"internal_v4_address_spec,omitempty"`
}

type MasterSpec_MasterType struct {
	// +optional
	ZonalMasterSpec *ZonalMasterSpec `json:"zonal_master_spec"`
	// +optional
	RegionalMasterSpec *RegionalMasterSpec `json:"regional_master_spec"`
}

type MasterMaintenancePolicy struct {
	// If set to true, automatic updates are installed in the specified period of time with no interaction from the user.
	// If set to false, automatic upgrades are disabled.
	// +kubebuilder:validation:Required
	AutoUpgrade bool `json:"auto_upgrade,omitempty"`
	// Maintenance window settings. Update will start at the specified time and last no more than the specified duration.
	// The time is set in UTC.
	// +optional
	MaintenanceWindow *MaintenanceWindow `json:"maintenance_window,omitempty"`
}

type MaintenanceWindow struct {
	// Maintenance policy.
	//
	// Types that are assignable to Policy:
	//	*MaintenanceWindow_Anytime
	//	*MaintenanceWindow_DailyMaintenanceWindow
	//	*MaintenanceWindow_WeeklyMaintenanceWindow
	// +optional
	Policy *MaintenanceWindow_Policy `json:"policy"`
}

type MaintenanceWindow_Policy struct {
	// Updating the master at any time.
	// +optional
	Anytime *AnytimeMaintenanceWindow `json:"anytime"`
	// Updating the master on any day during the specified time window.
	// +optional
	DailyMaintenanceWindow *DailyMaintenanceWindow `json:"daily_maintenance_window"`
	// Updating the master on selected days during the specified time window.
	// +optional
	WeeklyMaintenanceWindow *WeeklyMaintenanceWindow `json:"weekly_maintenance_window"`
}

type AnytimeMaintenanceWindow struct{}

type DailyMaintenanceWindow struct {
	// Window start time, in the UTC timezone.
	// +optional
	StartTime *TimeOfDay `json:"start_time,omitempty"`
	// Window duration.
	// +optional
	Duration *Duration `json:"duration,omitempty"`
}

type TimeOfDay struct {
	// Hours of day in 24 hour format. Should be from 0 to 23. An API may choose
	// to allow the value "24:00:00" for scenarios like business closing time.
	// +optional
	Hours int32 `json:"hours,omitempty"`
	// Minutes of hour of day. Must be from 0 to 59.
	// +optional
	Minutes int32 `json:"minutes,omitempty"`
	// Seconds of minutes of the time. Must normally be from 0 to 59. An API may
	// allow the value 60 if it allows leap-seconds.
	// +optional
	Seconds int32 `json:"seconds,omitempty"`
	// Fractions of seconds in nanoseconds. Must be from 0 to 999,999,999.
	// +optional
	Nanos int32 `json:"nanos,omitempty"`
}

type Duration struct {
	// Signed seconds of the span of time. Must be from -315,576,000,000
	// to +315,576,000,000 inclusive. Note: these bounds are computed from:
	// 60 sec/min * 60 min/hr * 24 hr/day * 365.25 days/year * 10000 years
	// +optional
	Seconds int64 `json:"seconds,omitempty"`
	// Signed fractions of a second at nanosecond resolution of the span
	// of time. Durations less than one second are represented with a 0
	// `seconds` field and a positive or negative `nanos` field. For durations
	// of one second or more, a non-zero value for the `nanos` field must be
	// of the same sign as the `seconds` field. Must be from -999,999,999
	// to +999,999,999 inclusive.
	// +optional
	Nanos int32 `json:"nanos,omitempty"`
}

type WeeklyMaintenanceWindow struct {
	// Days of the week and the maintenance window for these days when automatic updates are allowed.
	// +optional
	DaysOfWeek []*DaysOfWeekMaintenanceWindow `json:"days_of_week,omitempty"`
}

type DaysOfWeekMaintenanceWindow struct {
	// Days of the week when automatic updates are allowed.
	Days []string `json:"days,omitempty"`
	// Window start time, in the UTC timezone.
	// +optional
	StartTime *TimeOfDay `json:"start_time,omitempty"`
	// Window duration.
	// +optional
	Duration *Duration `json:"duration,omitempty"`
}

type MasterSpec struct {
	// Types that are assignable to MasterType:
	//	*MasterSpec_ZonalMasterSpec
	//	*MasterSpec_RegionalMasterSpec
	// +kubebuilder:validation:Required
	MasterType MasterSpec_MasterType `json:"master_type"`
	// Version of Kubernetes components that runs on the master.
	// +optional
	Version string `json:"version,omitempty"`
	// Maintenance policy of the master.
	// +optional
	MaintenancePolicy *MasterMaintenancePolicy `json:"maintenance_policy,omitempty"`
	// Master security groups.
	// +optional
	SecurityGroupIds []string `json:"security_group_ids,omitempty"`
}

type IPAllocationPolicy struct {
	// CIDR block. IP range for allocating pod addresses.
	//
	// It should not overlap with any subnet in the network the Kubernetes cluster located in. Static routes will be
	// set up for this CIDR blocks in node subnets.
	// +optional
	ClusterIpv4CidrBlock string `json:"cluster_ipv4_cidr_block,omitempty"`
	// Size of the masks that are assigned for each node in the cluster.
	//
	// If not specified, 24 is used.
	// +optional
	NodeIpv4CidrMaskSize int64 `json:"node_ipv4_cidr_mask_size,omitempty"`
	// CIDR block. IP range Kubernetes service Kubernetes cluster IP addresses will be allocated from.
	//
	// It should not overlap with any subnet in the network the Kubernetes cluster located in.
	// +optional
	ServiceIpv4CidrBlock string `json:"service_ipv4_cidr_block,omitempty"`
	// IPv6 range for allocating pod IP addresses.
	// +optional
	ClusterIpv6CidrBlock string `json:"cluster_ipv6_cidr_block,omitempty"`
	// IPv6 range for allocating Kubernetes service IP addresses
	// +optional
	ServiceIpv6CidrBlock string `json:"service_ipv6_cidr_block,omitempty"`
}

type NetworkPolicy struct {
	// +optional
	Provider string `json:"provider,omitempty"`
}

type KMSProvider struct {
	// KMS key ID for secrets encryption.
	// To obtain a KMS key ID use a [yandex.cloud.kms.v1.SymmetricKeyService.List] request.
	// +optional
	KeyId string `json:"key_id,omitempty"`
}

type Cluster_Cilium struct {
	// +optional
	Cilium *Cilium `json:"cilium"`
}

type Cilium struct {
	// +optional
	RoutingMode string `json:"routing_mode,omitempty"`
}

type Cluster_GatewayIpv4Address struct {
	// Gateway IPv4 address.
	// +optional
	GatewayIpv4Address string `json:"gateway_ipv4_address,omitempty"`
}

// ClusterParameters are the configurable fields of a Cluster.
type ClusterParameters struct {
	// ID of the folder to create an instance in.
	// To get the folder ID, use a [yandex.cloud.resourcemanager.v1.FolderService.List] request.
	// +kubebuilder:validation:Required
	FolderID string `json:"folder_id"`
	// Name of the instance.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Description of the instance.
	// +optional
	Description string `json:"description,omitempty"`
	// Resource labels as `key:value` pairs.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// ID of the availability zone where the instance resides.
	// To get a list of available zones, use the [yandex.cloud.compute.v1.ZoneService.List] request
	// +kubebuilder:validation:Required
	ZoneID string `json:"zone_id,omitempty"`
	// ID of the network.
	// +kubebuilder:validation:Required
	NetworkId string `json:"network_id,omitempty"`
	// IP allocation policy of the Kubernetes cluster.
	// +kubebuilder:validation:Required
	MasterSpec *MasterSpec `json:"master_spec,omitempty"`
	// IP allocation policy of the Kubernetes cluster.
	// +optional
	IpAllocationPolicy *IPAllocationPolicy `json:"ip_allocation_policy,omitempty"`
	// Types that are assignable to InternetGateway:
	//	*CreateClusterRequest_GatewayIpv4Address
	// For no there is only one implementation of gateway
	// +optional
	InternetGateway *Cluster_GatewayIpv4Address `json:"internet_gateway,omitempty"`
	// Service account to be used for provisioning Compute Cloud and VPC resources for Kubernetes cluster.
	// Selected service account should have `edit` role on the folder where the Kubernetes cluster will be
	// located and on the folder where selected network resides.
	// +kubebuilder:validation:Required
	ServiceAccountId string `json:"service_account_id,omitempty"`
	// Service account to be used by the worker nodes of the Kubernetes cluster to access Container Registry or to push node logs and metrics.
	// +kubebuilder:validation:Required
	NodeServiceAccountId string `json:"node_service_account_id,omitempty"`
	// Release channel for the master.
	// +kubebuilder:validation:Required
	ReleaseChannel string `json:"release_channel,omitempty"`
	// +optional
	NetworkPolicy *NetworkPolicy `json:"network_policy,omitempty"`
	// KMS provider configuration.
	// +optional
	KmsProvider *KMSProvider `json:"kms_provider,omitempty"`
	// Types that are assignable to NetworkImplementation:
	//	*CreateClusterRequest_Cilium
	// For now there is only one implementation
	// +optional
	NetworkImplementation *Cluster_Cilium `json:"network_implementation,omitempty"`
}

// ClusterObservation are the observable fields of a Cluster.
type ClusterObservation struct {
	ID          string            `json:"ID"`
	FolderID    string            `json:"folder_id"`
	CreatedAt   string            `json:"created_at"`
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Description string            `json:"description,omitempty"`
}

// An ClusterSpec defines the desired state of a Cluster.
type ClusterSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ClusterParameters `json:"forProvider"`
}

// An ClusterStatus represents the observed state of a Cluster.
type ClusterStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ClusterObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An Cluster is an API type for YC Kubernetes Cluster provider
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="YC_NAME",type="string",JSONPath=".status.atProvider.name"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,yc}
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Instance
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}
