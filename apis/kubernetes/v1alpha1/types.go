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
	SubnetID    string          `json:"subnetId,omitempty"`
	SubnetIDRef *xpv1.Reference `json:"subnetIdRef,omitempty"`
	// +optional
	SubnetIDSelector *xpv1.Selector `json:"subnetIdSelector,omitempty"`
}

type ExternalAddressSpec struct{}

type ZonalMasterSpec struct {
	// ID of the availability zone.
	// +optional
	ZoneId string `json:"zoneId,omitempty"`
	// Specification of parameters for internal IPv4 networking.
	// +optional
	InternalV4AddressSpec *InternalAddressSpec `json:"internalV4addressSpec,omitempty"`
	// Specification of parameters for external IPv4 networking.
	// +optional
	ExternalV4AddressSpec *ExternalAddressSpec `json:"externalV4AddressSpec,omitempty"`
}

type RegionalMasterSpec struct {
	// ID of the availability zone where the master resides.
	// +optional
	RegionId string `json:"regionId,omitempty"`
	// List of locations where the master will be allocated.
	// +optional
	Locations []*MasterLocation `json:"locations,omitempty"`
	// Specify to allocate a static public IP for the master.
	// +optional
	ExternalV4AddressSpec *ExternalAddressSpec `json:"externalV4AddressSpec,omitempty"`
}

type MasterLocation struct {
	// ID of the availability zone.
	// +optional
	ZoneId string `json:"zoneId,omitempty"`
	// If not specified and there is a single subnet in specified zone, address
	// in this subnet will be allocated.
	// +optional
	InternalV4AddressSpec *InternalAddressSpec `json:"internalV4AddressSpec,omitempty"`
}

type MasterSpec_MasterType struct {
	// +optional
	ZonalMasterSpec *ZonalMasterSpec `json:"zonalMasterSpec,omitempty"`
	// +optional
	RegionalMasterSpec *RegionalMasterSpec `json:"regionalMasterSpec,omitempty"`
}

type ZonalMaster struct {
	// ID of the availability zone where the master resides.
	ZoneId string `json:"zoneId,omitempty"`
	// IPv4 internal network address that is assigned to the master.
	// +optional
	InternalV4Address string `json:"internalV4Address,omitempty"`
	// IPv4 external network address that is assigned to the master.
	// +optional
	ExternalV4Address string `json:"externalV4Address,omitempty"`
}

type RegionalMaster struct {
	// ID of the region where the master resides.
	RegionId string `json:"regionId,omitempty"`
	// IPv4 internal network address that is assigned to the master.
	// +optional
	InternalV4Address string `json:"internalV4Address,omitempty"`
	// IPv4 external network address that is assigned to the master.
	// +optional
	ExternalV4Address string `json:"externalV4Address,omitempty"`
}

type Master_MasterType struct {
	// Parameters of the availability zone for the master.
	// +optional
	ZonalMaster *ZonalMaster `json:"zonalMaster,omitempty"`
	// +optional
	RegionalMaster *RegionalMaster `json:"regionalMaster,omitempty"`
}

type Master struct {
	MasterType *Master_MasterType `json:"masterType"`
	// Version of Kubernetes components that runs on the master.
	Version string `json:"version,omitempty"`
	// Endpoints of the master. Endpoints constitute of scheme and port (i.e. `https://ip-address:port`)
	//and can be used by the clients to communicate with the Kubernetes API of the Kubernetes cluster.
	// TODO: if needed
	// Endpoints *MasterEndpoints `protobuf:"bytes,3,opt,name=endpoints,proto3" json:"endpoints,omitempty"`
	// Master authentication parameters are used to establish trust between the master and a client.
	// TODO: if needed
	// MasterAuth *MasterAuth `protobuf:"bytes,4,opt,name=master_auth,json=masterAuth,proto3" json:"master_auth,omitempty"`
	// Detailed information about the Kubernetes version that is running on the master.
	// TODO: if needed
	// VersionInfo *VersionInfo `protobuf:"bytes,5,opt,name=version_info,json=versionInfo,proto3" json:"version_info,omitempty"`
	// Maintenance policy of the master.
	MaintenancePolicy *MasterMaintenancePolicy `json:"maintenancePolicy,omitempty"`
	// Master security groups.
	// +optional
	SecurityGroupIds []string `json:"securityGroupIds,omitempty"`
}

type MasterMaintenancePolicy struct {
	// If set to true, automatic updates are installed in the specified period of time with no interaction from the user.
	// If set to false, automatic upgrades are disabled.
	// +kubebuilder:validation:Required
	AutoUpgrade bool `json:"autoUpgrade,omitempty"`
	// Maintenance window settings. Update will start at the specified time and last no more than the specified duration.
	// The time is set in UTC.
	// +optional
	MaintenanceWindow *MaintenanceWindow `json:"maintenanceWindow,omitempty"`
}

type MaintenanceWindow struct {
	// Maintenance policy.
	//
	// Types that are assignable to Policy:
	//	*MaintenanceWindow_Anytime
	//	*MaintenanceWindow_DailyMaintenanceWindow
	//	*MaintenanceWindow_WeeklyMaintenanceWindow
	// +kubebuilder:validation:Required
	Policy *MaintenanceWindow_Policy `json:"policy,omitempty"`
}

type MaintenanceWindow_Policy struct {
	// Updating the master at any time.
	// +optional
	Anytime *AnytimeMaintenanceWindow `json:"anytime,omitempty"`
	// Updating the master on any day during the specified time window.
	// +optional
	DailyMaintenanceWindow *DailyMaintenanceWindow `json:"dailyMaintenanceWindow,omitempty"`
	// Updating the master on selected days during the specified time window.
	// +optional
	WeeklyMaintenanceWindow *WeeklyMaintenanceWindow `json:"weeklyMaintenanceWindow,omitempty"`
}

type AnytimeMaintenanceWindow struct{}

type DailyMaintenanceWindow struct {
	// Window start time, in the UTC timezone.
	// +optional
	StartTime *TimeOfDay `json:"startTime,omitempty"`
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
	// +kubebuilder:validation:Minimum:=3600
	// +kubebuilder:validation:Maximum:=86400
	// +kubebuilder:validation:Required
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
	// +kubebuilder:validation:Required
	DaysOfWeek []*DaysOfWeekMaintenanceWindow `json:"daysOfWeek,omitempty"`
}

type DaysOfWeekMaintenanceWindow struct {
	// Days of the week when automatic updates are allowed.
	// +kubebuilder:validation:Required
	Days []string `json:"days,omitempty"`
	// Window start time, in the UTC timezone.
	// +kubebuilder:validation:Required
	StartTime *TimeOfDay `json:"startTime,omitempty"`
	// Window duration.
	// +kubebuilder:validation:Required
	Duration *Duration `json:"duration,omitempty"`
}

type MasterSpec struct {
	// Types that are assignable to MasterType:
	//	*MasterSpec_ZonalMasterSpec
	//	*MasterSpec_RegionalMasterSpec
	// +kubebuilder:validation:Required
	MasterType *MasterSpec_MasterType `json:"masterType"`
	// Version of Kubernetes components that runs on the master.
	// +kubebuilder:validation:Required
	Version string `json:"version"`
	// Maintenance policy of the master.
	// +optional
	MaintenancePolicy *MasterMaintenancePolicy `json:"maintenancePolicy,omitempty"`
	// Master security groups.
	// +optional
	SecurityGroupIds []string `json:"securityGroupIds,omitempty"`
}

type IPAllocationPolicy struct {
	// CIDR block. IP range for allocating pod addresses.
	//
	// It should not overlap with any subnet in the network the Kubernetes cluster located in. Static routes will be
	// set up for this CIDR blocks in node subnets.
	// +kubebuilder:validation:Required
	ClusterIpv4CidrBlock string `json:"clusterIpv4CidrBlock,omitempty"`
	// Size of the masks that are assigned for each node in the cluster.
	//
	// If not specified, 24 is used.
	// +kubebuilder:validation:Required
	NodeIpv4CidrMaskSize int64 `json:"nodeIpv4CidrMaskSize,omitempty"`
	// CIDR block. IP range Kubernetes service Kubernetes cluster IP addresses will be allocated from.
	//
	// It should not overlap with any subnet in the network the Kubernetes cluster located in.
	// +kubebuilder:validation:Required
	ServiceIpv4CidrBlock string `json:"serviceIpv4CidrBlock,omitempty"`
	// IPv6 range for allocating pod IP addresses.
	// +optional
	ClusterIpv6CidrBlock string `json:"clusterIpv6CidrBlock,omitempty"`
	// IPv6 range for allocating Kubernetes service IP addresses
	// +optional
	ServiceIpv6CidrBlock string `json:"serviceIpv6CidrBlock,omitempty"`
}

type NetworkPolicy struct {
	// +kubebuilder:validation:Required
	Provider string `json:"provider,omitempty"`
}

type KMSProvider struct {
	// KMS key ID for secrets encryption.
	// To obtain a KMS key ID use a [yandex.cloud.kms.v1.SymmetricKeyService.List] request.
	// +kubebuilder:validation:Required
	KeyId string `json:"keyId"`
}

type Cluster_Cilium struct {
	// +kubebuilder:validation:Required
	Cilium *Cilium `json:"cilium"`
}

type Cilium struct {
	// +kubebuilder:validation:Required
	RoutingMode string `json:"routingMode"`
}

type Cluster_GatewayIpv4Address struct {
	// Gateway IPv4 address.
	// +kubebuilder:validation:Required
	GatewayIpv4Address string `json:"gatewayIpv4Address"`
}

type MasterUpdateSpec struct {
	// Specification of the master update.
	// +optional
	Version *UpdateVersionSpec `json:"version,omitempty"`
	// Maintenance policy of the master.
	// +optional
	MaintenancePolicy *MasterMaintenancePolicy `json:"maintenancePolicy,omitempty"`
	// Master security groups.
	// +optional
	SecurityGroupIds []string `json:"securityGroupIds,omitempty"`
}

type UpdateVersionSpec struct {
	// Types that are assignable to Specifier:
	//	*UpdateVersionSpec_Version
	//	*UpdateVersionSpec_LatestRevision
	// +kubebuilder:validation:Required
	Specifier *Specifier `json:"specifier"`
}

type Specifier struct {
	// Request update to a newer version of Kubernetes (1.x -> 1.y).
	// +optional
	Version string `json:"version,omitempty"`
	// Request update to the latest revision for the current version.
	// +optional
	LatestRevision bool `json:"latestRevision,omitempty"`
}

// ClusterObservation are the observable fields of a Cluster.
type ClusterObservation struct {
	ID                   string                      `json:"ID"`
	FolderID             string                      `json:"folderId"`
	CreatedAt            string                      `json:"createdAt"`
	Name                 string                      `json:"name"`
	Labels               map[string]string           `json:"labels,omitempty"`
	Description          string                      `json:"description,omitempty"`
	Status               string                      `json:"status,omitempty"`
	Health               string                      `json:"health,omitempty"`
	ServiceAccountId     string                      `json:"serviceAccount_id,omitempty"`
	NodeServiceAccountId string                      `json:"nodeService_account_id,omitempty"`
	InternetGateway      *Cluster_GatewayIpv4Address `json:"internetGateway,omitempty"`
	NetworkPolicy        *NetworkPolicy              `json:"networkPolicy,omitempty"`
	Master               *Master                     `json:"master,omitempty"`
}

// An ClusterSpec defines the desired state of a Cluster.
type ClusterSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	// ID of the folder to create an instance in.
	// To get the folder ID, use a [yandex.cloud.resourcemanager.v1.FolderService.List] request.
	// +kubebuilder:validation:Required
	FolderID string `json:"folderId"`
	// Description of the instance.
	// +optional
	Description string `json:"description,omitempty"`
	// Resource labels as `key:value` pairs.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// ID of the availability zone where the instance resides.
	// To get a list of available zones, use the [yandex.cloud.compute.v1.ZoneService.List] request
	// +kubebuilder:validation:Required
	ZoneID string `json:"zoneId,omitempty"`
	// Name of the network.
	// +optional
	NetworkID string `json:"networkId,omitempty"`
	// +optional
	NetworkIDRef *xpv1.Reference `json:"networkIdRef,omitempty"`
	// +optional
	NetworkIDSelector *xpv1.Selector `json:"networkIdSelector,omitempty"`
	// IP allocation policy of the Kubernetes cluster.
	// +kubebuilder:validation:Required
	MasterSpec *MasterSpec `json:"masterSpec"`
	// IP allocation policy of the Kubernetes cluster.
	// +optional
	IpAllocationPolicy *IPAllocationPolicy `json:"ipAllocationPolicy,omitempty"`
	// Types that are assignable to InternetGateway:
	//	*CreateClusterRequest_GatewayIpv4Address
	// For no there is only one implementation of gateway
	// +optional
	InternetGateway *Cluster_GatewayIpv4Address `json:"internetGateway,omitempty"`
	// Service account to be used for provisioning Compute Cloud and VPC resources for Kubernetes cluster.
	// Selected service account should have `edit` role on the folder where the Kubernetes cluster will be
	// located and on the folder where selected network resides.
	// +kubebuilder:validation:Required
	ServiceAccountName string `json:"serviceAccountName"`
	// Service account to be used by the worker nodes of the Kubernetes cluster to access Container Registry or to push node logs and metrics.
	// +kubebuilder:validation:Required
	NodeServiceAccountName string `json:"nodeServiceAccountName"`
	// Release channel for the master.
	// +kubebuilder:validation:Required
	ReleaseChannel string `json:"releaseChannel"`
	// +optional
	NetworkPolicy *NetworkPolicy `json:"networkPolicy,omitempty"`
	// KMS provider configuration.
	// +optional
	KmsProvider *KMSProvider `json:"kmsProvider,omitempty"`
	// Types that are assignable to NetworkImplementation:
	//	*CreateClusterRequest_Cilium
	// For now there is only one implementation
	// +optional
	NetworkImplementation *Cluster_Cilium `json:"networkImplementation,omitempty"`
}

// An ClusterStatus represents the observed state of a Cluster.
type ClusterStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ClusterObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An Cluster is an API type for YC Kubernetes Cluster provider
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="HEALTH",type="string",JSONPath=".status.atProvider.health"
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

// NodeGroup
type NodeGroupSpec struct {
	xpv1.ResourceSpec `json:",inline"`

	FolderID string `json:"folderId"`
	// ID of the Kubernetes cluster to create a node group in.
	// To get the Kubernetes cluster ID, use a [ClusterService.List] request.
	// +optional
	ClusterId string `json:"clusterId,omitempty"`
	// +optional
	ClusterIdRef *xpv1.Reference `json:"clusterIdRef,omitempty"`
	// +optional
	ClusterIdSelector *xpv1.Selector `json:"clusterIdSelector,omitempty"`
	// Description of the node group.
	// +optional
	Description string `json:"description,omitempty"`
	// Resource labels as `key:value` pairs.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Node template for creating the node group.
	// +kubebuilder:validation:Required
	NodeTemplate *NodeTemplate `json:"nodeTemplate"`
	// Scale policy of the node group.
	// +kubebuilder:validation:Required
	ScalePolicy *ScalePolicy `json:"scalePolicy"`
	// Allocation policy of the node group by the zones and regions.
	// +kubebuilder:validation:Required
	AllocationPolicy *NodeGroupAllocationPolicy `json:"allocationPolicy"`
	// Deploy policy according to which the updates are rolled out. If not specified,
	// the default is used.
	// +optional
	// +kubebuilder:validation:Required
	DeployPolicy *DeployPolicy `json:"deployPolicy"`
	// Version of Kubernetes components that runs on the nodes.
	// +kubebuilder:validation:Required
	Version string `json:"version"`
	// Maintenance policy of the node group.
	// +kubebuilder:validation:Required
	MaintenancePolicy *NodeGroupMaintenancePolicy `json:"maintenancePolicy"`
	// Support for unsafe sysctl parameters. For more details see [documentation](https://kubernetes.io/docs/tasks/administer-cluster/sysctl-cluster/).
	// +optional
	AllowedUnsafeSysctls []string `json:"allowedUnsafeSysctls,omitempty"`
	// Taints that are applied to the nodes of the node group at creation time.
	// +optional
	NodeTaints Taints `json:"nodeTaints,omitempty"`
	// Labels that are assigned to the nodes of the node group at creation time.
	// +optional
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`
}

type NodeTemplate struct {
	// ID of the hardware platform configuration for the node.
	// +kubebuilder:validation:Required
	PlatformId string `json:"platformId"`
	// Computing resources of the node such as the amount of memory and number of cores.
	// +kubebuilder:validation:Required
	ResourcesSpec *ResourcesSpec `json:"resourcesSpec"`
	// Specification for the boot disk that will be attached to the node.
	// +kubebuilder:validation:Required
	BootDiskSpec *DiskSpec `json:"bootDiskSpec"`
	// The metadata as `key:value` pairs assigned to this instance template. This includes custom metadata and predefined keys.
	//
	// For example, you may use the metadata in order to provide your public SSH key to the node.
	// For more information, see [Metadata](/docs/compute/concepts/vm-metadata).
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
	// Scheduling policy configuration.
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"schedulingPolicy,omitempty"`
	// New api, to specify network interfaces for the node group compute instances.
	// Can not be used together with 'v4_address_spec'
	// +optional
	NetworkInterfaceSpecs NetworkInterfaceSpecs `json:"networkInterfaceSpecs,omitempty"`
	// +optional
	PlacementPolicy *PlacementPolicy `json:"placementPolicy,omitempty"`
	// this parameter allows to specify type of network acceleration used on nodes (instances)
	// +optional
	NetworkSettings *NodeTemplate_NetworkSettings `json:"networkSettings,omitempty"`
}

type ResourcesSpec struct {
	// Amount of memory available to the node, specified in bytes.
	// +optional
	Memory int64 `json:"memory,omitempty"`
	// Number of cores available to the node.
	// +optional
	Cores int64 `json:"cores,omitempty"`
	// Baseline level of CPU performance with the possibility to burst performance above that baseline level.
	// This field sets baseline performance for each core.
	// +optional
	CoreFraction int64 `json:"coreFraction,omitempty"`
	// Number of GPUs available to the node.
	// +optional
	Gpus int64 `json:"gpus,omitempty"`
}

type DiskSpec struct {
	// ID of the disk type.
	// +kubebuilder:validation:Required
	DiskTypeId string `json:"diskTypeId"`
	// Size of the disk, specified in bytes.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=32212254720
	DiskSize int64 `json:"diskSize"`
}

type NodeAddressSpec struct {
	// One-to-one NAT configuration. Setting up one-to-one NAT ensures that public IP addresses are assigned to nodes, and therefore internet is accessible for all nodes of the node group. If the field is not set, NAT will not be set up.
	// +kubebuilder:validation:Required
	OneToOneNatSpec *OneToOneNatSpec `json:"oneToOneNatSpec"`
}

type OneToOneNatSpec struct {
	// IP version for the public IP address.
	// +kubebuilder:validation:Required
	IpVersion string `json:"ipVersion"`
}

type SchedulingPolicy struct {
	// True for preemptible compute instances. Default value is false. Preemptible compute instances are stopped at least once every 24 hours, and can be stopped at any time
	// if their resources are needed by Compute.
	// For more information, see [Preemptible Virtual Machines](/docs/compute/concepts/preemptible-vm).
	// +kubebuilder:validation:Required
	Preemptible bool `json:"preemptible"`
}

type NetworkInterfaceSpecs []*NetworkInterfaceSpec

// TODO: need work with references
type NetworkInterfaceSpec struct {
	// IDs of the subnets.
	// +optional
	SubnetIds []string `json:"subnetIds,omitempty"`
	// +optional
	SubnetIdsRefs []*xpv1.Reference `json:"subnetIdsRefs,omitempty"`
	// +optional
	SubnetIdsSelectors []*xpv1.Selector `json:"subnetIdsSelectors,omitempty"`
	// Primary IPv4 address that is assigned to the instance for this network interface.
	// +optional
	PrimaryV4AddressSpec *NodeAddressSpec `json:"primaryV4addressSpec,omitempty"`
	// Primary IPv6 address that is assigned to the instance for this network interface.
	// +optional
	PrimaryV6AddressSpec *NodeAddressSpec `json:"primaryV6addressSpec,omitempty"`
	// IDs of security groups.
	// +optional
	SecurityGroupIds []string `json:"securityGroupIds,omitempty"`
}

type PlacementPolicy struct {
	// Identifier of placement group
	// +kubebuilder:validation:Required
	PlacementGroupId string `json:"placementGroupId"`
}

type NodeTemplate_NetworkSettings struct {
	// +kubebuilder:validation:Required
	Type string `json:"type"`
}

type ScalePolicy struct {
	// Types that are assignable to ScaleType:
	//	*ScalePolicy_FixedScale_
	//	*ScalePolicy_AutoScale_
	// +kubebuilder:validation:Required
	ScaleType *IsScalePolicy_ScaleType `json:"scaleType"`
}

type IsScalePolicy_ScaleType struct {
	// Fixed scale policy of the node group.
	// +optional
	FixedScale *ScalePolicy_FixedScale `json:"fixedScale,omitempty"`
	// Auto scale policy of the node group.
	// +optional
	AutoScale *ScalePolicy_AutoScale `json:"autoScale,omitempty"`
}

type ScalePolicy_FixedScale struct {
	// Number of nodes in the node group.
	// +kubebuilder:validation:Required
	Size int64 `json:"size"`
}

type ScalePolicy_AutoScale struct {
	// Minimum number of nodes in the node group.
	// +kubebuilder:validation:Required
	MinSize int64 `json:"minSize"`
	// Maximum number of nodes in the node group.
	// +kubebuilder:validation:Required
	MaxSize int64 `json:"maxSize"`
	// Initial number of nodes in the node group.
	// +kubebuilder:validation:Required
	InitialSize int64 `json:"initialSize"`
}

type NodeGroupAllocationPolicy struct {
	// List of locations where resources for the node group will be allocated.
	// +kubebuilder:validation:Required
	Locations NodeGroupLocations `json:"locations"`
}

type NodeGroupLocations []*NodeGroupLocation

type NodeGroupLocation struct {
	// ID of the availability zone where the nodes may reside.
	// TODO: should be reference probably
	// +kubebuilder:validation:Required
	ZoneId string `json:"zoneId"`
	// ID of the subnet. If a network chosen for the Kubernetes cluster has only one subnet in the specified zone, subnet ID may be omitted.
	// +optional
	SubnetId string `json:"subnetId,omitempty"`
	// +optional
	SubnetIdRef *xpv1.Reference `json:"subnetIdRef,omitempty"`
	// +optional
	SubnetIdSelector *xpv1.Selector `json:"subnetIdSelector,omitempty"`
}

type DeployPolicy struct {
	// The maximum number of running instances that can be taken offline (i.e.,
	// stopped or deleted) at the same time during the update process.
	// If [max_expansion] is not specified or set to zero, [max_unavailable] must
	// be set to a non-zero value.
	// +kubebuilder:validation:Required
	MaxUnavailable int64 `json:"maxUnavailable"`
	// The maximum number of instances that can be temporarily allocated above
	// the group's target size during the update process.
	// If [max_unavailable] is not specified or set to zero, [max_expansion] must
	// be set to a non-zero value.
	// +kubebuilder:validation:Required
	MaxExpansion int64 `json:"maxExpansion"`
}

type NodeGroupMaintenancePolicy struct {
	// If set to true, automatic updates are installed in the specified period of time with no interaction from the user.
	// If set to false, automatic upgrades are disabled.
	// +optional
	AutoUpgrade bool `json:"autoUpgrade,omitempty"`
	// If set to true, automatic repairs are enabled. Default value is false.
	// +optional
	AutoRepair bool `json:"autoRepair,omitempty"`
	// Maintenance window settings. Update will start at the specified time and last no more than the specified duration.
	// The time is set in UTC.
	// +optional
	MaintenanceWindow *MaintenanceWindow `json:"maintenanceWindow,omitempty"`
}

type Taint struct {
	// The taint key to be applied to a node.
	// +kubebuilder:validation:Required
	Key string `json:"key"`
	// The taint value corresponding to the taint key.
	// +kubebuilder:validation:Required
	Value string `json:"value"`
	// The effect of the taint on pods that do not tolerate the taint.
	// +kubebuilder:validation:Required
	Effect string `json:"effect"`
}

type Taints []*Taint

type VersionInfo struct {
	// Current Kubernetes version, format: major.minor (e.g. 1.15).
	CurrentVersion string `json:"currentVersion,omitempty"`
	// Newer revisions may include Kubernetes patches (e.g 1.15.1 -> 1.15.2) as well
	// as some internal component updates - new features or bug fixes in Yandex specific
	// components either on the master or nodes.
	NewRevisionAvailable bool `json:"newRevisionAvailable,omitempty"`
	// Description of the changes to be applied when updating to the latest
	// revision. Empty if new_revision_available is false.
	NewRevisionSummary string `json:"newRevisionSummary,omitempty"`
	// The current version is on the deprecation schedule, component (master or node group)
	// should be upgraded.
	VersionDeprecated bool `json:"versionDeprecated,omitempty"`
}

// An NodeGroupStatus represents the observed state of a NodeGroup.
type NodeGroupStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          NodeGroupObservation `json:"atProvider,omitempty"`
}

type NodeGroupObservation struct {
	ID                   string                      `json:"id,omitempty"`
	ClusterID            string                      `json:"clusterId,omitempty"`
	Name                 string                      `json:"name,omitempty"`
	Status               string                      `json:"status,omitempty"`
	CreatedAt            string                      `json:"createdAt,omitempty"`
	Description          string                      `json:"description,omitempty"`
	Labels               map[string]string           `json:"labels,omitempty"`
	NodeTemplate         *NodeTemplate               `json:"nodeTemplate,omitempty"`
	ScalePolicy          *ScalePolicy                `json:"scalePolicy,omitempty"`
	AllocationPolicy     *NodeGroupAllocationPolicy  `json:"allocationPolicy,omitempty"`
	DeployPolicy         *DeployPolicy               `json:"deployPolicy,omitempty"`
	VersionInfo          *VersionInfo                `json:"versionInfo,omitempty"`
	MaintenancePolicy    *NodeGroupMaintenancePolicy `json:"maintenancePolicy,omitempty"`
	AllowedUnsafeSysctls []string                    `json:"allowedUnsafeSysctls,omitempty"`
	NodeTaints           Taints                      `json:"nodeTaints,omitempty"`
	NodeLabels           map[string]string           `json:"nodeLabels,omitempty"`
}

// +kubebuilder:object:root=true

// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".status.atProvider.name"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,yc}
type NodeGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeGroupSpec   `json:"spec"`
	Status NodeGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Instance
type NodeGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeGroup `json:"items"`
}
