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

// ResourcesSpec describes resources of compute instance
type ResourcesSpec struct {
	// The amount of memory available to the instance, specified in bytes.
	Memory int64 `json:"memory"`
	// The number of cores available to the instance.
	Cores int64 `json:"cores"`
	// Baseline level of CPU performance with the ability to burst performance above that baseline level.
	// This field sets baseline performance for each core.
	// +optional
	CoreFraction int64 `json:"core_fraction,omitempty"`
	// The number of GPUs available to the instance.
	// +optional
	Gpus int64 `json:"gpus,omitempty"`
}

// +kubebuilder:validation:Enum=Allow;Forbid;Replace
type AttachedDiskMode int32

const (
	// Unspecified
	AttachedDiskSpec_MODE_UNSPECIFIED AttachedDiskMode = 0
	// Read-only access.
	AttachedDiskSpec_READ_ONLY AttachedDiskMode = 1
	// Read/Write access. Default value.
	AttachedDiskSpec_READ_WRITE AttachedDiskMode = 2
)

type DiskPlacementPolicy struct {
	PlacementGroupID string `json:"placement_group_id,omitempty"`
}

type SourceSpec struct {
	// Should be one of, as I understand
	// +optional
	ImageID string `json:"image_id,omitempty"`
	// +optional
	SnapshotID string `json:"snapshot_id,omitempty"`
}

type DiskSpec struct {
	// +optional
	ID string `json:"id,omitempty"`
	// Name of the disk.
	// +optional
	Name string `json:"name,omitempty"`
	// Description of the disk.
	// +optional
	Description string `json:"description,omitempty"`
	// ID of the disk type.
	// To get a list of available disk types, use the [yandex.cloud.compute.v1.DiskTypeService.List] request.
	// +optional
	TypeId string `json:"type_id,omitempty"`
	// Size of the disk, specified in bytes.
	Size int64 `json:"size,omitempty"`
	// Block size of the disk, specified in bytes. The default is 4096.
	// +optional
	BlockSize int64 `json:"block_size,omitempty"`
	// Placement policy configuration.
	// +optional
	DiskPlacementPolicy *DiskPlacementPolicy `json:"disk_placement_policy,omitempty"`
	// Types that are assignable to Source:
	//  *AttachedDiskSpec_DiskSpec_ImageId
	//  *AttachedDiskSpec_DiskSpec_SnapshotId
	Source *SourceSpec `json:"source"`
}

type AttachedDiskSpec struct {
	// +optional
	Mode string `json:"mode,omitempty"`
	// Specifies a unique serial number of your choice that is reflected into the /dev/disk/by-id/ tree
	// of a Linux operating system running within the instance.
	//
	// This value can be used to reference the device for mounting, resizing, and so on, from within the instance.
	// If not specified, a random value will be generated.
	// +optional
	DeviceName string `json:"device_name,omitempty"`
	// Specifies whether the disk will be auto-deleted when the instance is deleted.
	// +optional
	AutoDelete bool `json:"auto_delete,omitempty"`
	// Types that are assignable to Disk:
	//	*AttachedDiskSpec_DiskSpec_
	//	*AttachedDiskSpec_DiskId

	Disk *DiskSpec `json:"disk"`
}

type DnsRecord struct {
	// Name of the A/AAAA record as specified when creating the instance.
	// Note that if `fqdn' has no trailing '.', it is specified relative to the zone (@see dns_zone_id).
	Fqdn string `json:"fqdn,omitempty"`
	// DNS zone id for the record (optional, if not set, some private zone is used).
	DnsZoneId string `json:"dns_zone_id,omitempty"`
	// DNS record ttl (optional, if not set, a reasonable default is used.)
	Ttl int64 `json:"ttl,omitempty"`
	// When true, indicates there is a corresponding auto-created PTR DNS record.
	Ptr bool `json:"ptr,omitempty"`
}

type OneToOneNatSpec struct {
	// An external IP address associated with this instance.
	// +optional
	Address string `json:"address,omitempty"`
	// IP version for the external IP address.
	IpVersion string `json:"ip_version,omitempty"`
	// External DNS configuration
	// +optional
	DnsRecords []*DnsRecord `json:"dns_records,omitempty"`
}

type PrimaryAddressSpec struct {
	// An IPv4 internal network address that is assigned to the instance for this network interface.
	// If not specified by the user, an unused internal IP is assigned by the system.
	// +optional
	Address string `json:"address,omitempty"` // optional, manual set static int
	// An external IP address configuration.
	// If not specified, then this instance will have no external internet access.
	// +optional
	OneToOneNatSpec *OneToOneNatSpec `json:"one_to_one_nat"`
	// Internal DNS configuration
	// +optional
	DnsRecords []*DnsRecord `json:"dns_records,omitempty"`
}

type SchedulingPolicy struct {
	Preemptible bool `json:"preemptible,omitempty"`
}

type NetworkSettings struct {
	Type string `json:"type,omitempty"`
}

type PlacementPolicy_HostAffinityRule struct {
	// Affinity label or one of reserved values - 'yc.hostId', 'yc.hostGroupId'
	Key string `json:"key,omitempty"`
	// Include or exclude action
	Op string `json:"op,omitempty"`
	// Affinity value or host ID or host group ID
	Values []string `json:"values,omitempty"`
}

type PlacementPolicy struct {
	// Placement group ID.
	PlacementGroupID string `json:"placement_group_id,omitempty"`
	// List of affinity rules. Scheduler will attempt to allocate instances according to order of rules.
	HostAffinityRules []*PlacementPolicy_HostAffinityRule `json:"host_affinity_rules,omitempty"`
}

type NetworkInterfaceSpec struct {
	// +optional
	SubnetID   string `json:"subnet_id"`
	SubnetName string `json:"subnet_name"`
	// Primary IPv4 address that is assigned to the instance for this network interface.
	// +optional
	PrimaryV4Address *PrimaryAddressSpec `json:"primary_v4_address,omitempty"`
	// Primary IPv6 address that is assigned to the instance for this network interface. IPv6 not available yet.
	// +optional
	PrimaryV6Address *PrimaryAddressSpec `json:"primary_v6_address,omitempty"`
	// ID's of security groups attached to the interface
	// +optional
	SecurityGroupIds []string `json:"security_group_ids,omitempty"`
}

// InstanceParameters are the configurable fields of a Instance.
// This looks bad. Try use pb from go-sdk
type InstanceParameters struct {
	// ID of the folder to create an instance in.
	// To get the folder ID, use a [yandex.cloud.resourcemanager.v1.FolderService.List] request.
	FolderID string `json:"folder_id"`
	// Name of the instance.
	Name string `json:"name"`
	// Description of the instance.
	// +optional
	Description string `json:"description,omitempty"`
	// Resource labels as `key:value` pairs.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// ID of the availability zone where the instance resides.
	// To get a list of available zones, use the [yandex.cloud.compute.v1.ZoneService.List] request
	ZoneID string `json:"zone_id,omitempty"`
	// ID of the hardware platform configuration for the instance.
	// This field affects the available values in [resources_spec] field.
	//
	// Platforms allows you to create various types of instances: with a large amount of memory,
	// with a large number of cores, with a burstable performance.
	// For more information, see [Platforms](/docs/compute/concepts/vm-platforms).
	PlatformID string `json:"platform_id"`
	// Computing resources of the instance, such as the amount of memory and number of cores.
	// To get a list of available values, see [Levels of core performance](/docs/compute/concepts/performance-levels).
	Resources *ResourcesSpec `json:"resources"` // was pointer
	// The metadata `key:value` pairs that will be assigned to this instance. This includes custom metadata and predefined keys.
	// The total size of all keys and values must be less than 512 KB.
	//
	// Values are free-form strings, and only have meaning as interpreted by the programs which configure the instance.
	// The values must be 256 KB or less.
	//
	// For example, you may use the metadata in order to provide your public SSH key to the instance.
	// For more information, see [Metadata](/docs/compute/concepts/vm-metadata).
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
	// Boot disk to attach to the instance.
	BootDiskSpec *AttachedDiskSpec `json:"boot_disk,omitempty"`
	// Array of secondary disks to attach to the instance.
	// +optional
	SecondaryDisks []*AttachedDiskSpec `json:"secondary_disks,omitempty"`
	// Network configuration for the instance. Specifies how the network interface is configured
	// to interact with other services on the internal network and on the internet.
	// Currently only one network interface is supported per instance.
	NetworkInterfaces []*NetworkInterfaceSpec `json:"network_interfaces,omitempty"`
	// Host name for the instance.
	// This field is used to generate the [yandex.cloud.compute.v1.Instance.fqdn] value.
	// The host name must be unique within the network and region.
	// If not specified, the host name will be equal to [yandex.cloud.compute.v1.Instance.id] of the instance
	// and FQDN will be `<id>.auto.internal`. Otherwise FQDN will be `<hostname>.<region_id>.internal`.
	// +optional
	Hostname string `json:"hostname,omitempty"`
	// Scheduling policy configuration.
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"scheduling_policy,omitempty"`
	// ID of the service account to use for [authentication inside the instance](/docs/compute/operations/vm-connect/auth-inside-vm).
	// To get the service account ID, use a [yandex.cloud.iam.v1.ServiceAccountService.List] request.
	// +optional
	ServiceAccountID string `json:"service_account_id,omitempty"`
	// Network settings.
	// +optional
	NetworkSettings *NetworkSettings `json:"network_settings,omitempty"`
	// Placement policy configuration.
	// +optional
	PlacementPolicy *PlacementPolicy `json:"placement_policy,omitempty"`
}

// InstanceObservation are the observable fields of a Instance.
type InstanceObservation struct {
	ID               string            `json:"ID"`
	FolderID         string            `json:"folder_id"`
	CreatedAt        string            `json:"created_at"`
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	Labels           map[string]string `json:"labels,omitempty"`
	Description      string            `json:"description,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	PlatformID       string            `json:"platform_id,omitempty"`
	ServiceAccountID string            `json:"service_account_id,omitempty"`
	Resources        *ResourcesSpec    `json:"resources,omitempty"`
	NetworkSettings  *NetworkSettings  `json:"network_settings,omitempty"`
	PlacementPolicy  *PlacementPolicy  `json:"placement_policy,omitempty"`
}

// An InstanceSpec defines the desired state of a Instance.
type InstanceSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       InstanceParameters `json:"forProvider"`
}

// An InstanceStatus represents the observed state of a Instance.
type InstanceStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          InstanceObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An Instance is an API type for YC provider
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.ID"
// +kubebuilder:printcolumn:name="YC_NAME",type="string",JSONPath=".status.atProvider.name"
// +kubebuilder:printcolumn:name="FOLDER_ID",type="string",JSONPath=".status.atProvider.folder_id"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,yc}
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceSpec   `json:"spec"`
	Status InstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InstanceList contains a list of Instance
type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instance `json:"items"`
}
