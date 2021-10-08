package kubernetes

import (
	"github.com/RealFatCat/provider-yc/apis/kubernetes/v1alpha1"
	k8s_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
)

func NewTaints(t v1alpha1.Taints) []*k8s_pb.Taint {
	res := []*k8s_pb.Taint{}
	for _, i := range t {
		v := NewTaint(i)
		if v != nil {
			res = append(res, v)
		}
	}
	return res
}

func NewTaintsSDK(t []*k8s_pb.Taint) v1alpha1.Taints {
	res := v1alpha1.Taints{}
	for _, i := range t {
		v := NewTaintSDK(i)
		if v != nil {
			res = append(res, v)
		}
	}
	return res
}

func NewTaint(t *v1alpha1.Taint) *k8s_pb.Taint {
	if t != nil {
		var ef k8s_pb.Taint_Effect
		val, ok := k8s_pb.Taint_Effect_value[t.Effect]
		if ok {
			ef = k8s_pb.Taint_Effect(val)
		} else {
			ef = k8s_pb.Taint_EFFECT_UNSPECIFIED
		}
		return &k8s_pb.Taint{
			Key:    t.Key,
			Value:  t.Value,
			Effect: ef,
		}
	}
	return nil
}

func NewTaintSDK(t *k8s_pb.Taint) *v1alpha1.Taint {
	if t != nil {
		return &v1alpha1.Taint{
			Key:    t.Key,
			Value:  t.Value,
			Effect: t.Effect.String(),
		}
	}
	return nil
}

func NewNodeGroupMaintenancePolicy(ngmp *v1alpha1.NodeGroupMaintenancePolicy) *k8s_pb.NodeGroupMaintenancePolicy {
	if ngmp != nil {
		res := &k8s_pb.NodeGroupMaintenancePolicy{
			AutoUpgrade: ngmp.AutoUpgrade,
			AutoRepair:  ngmp.AutoRepair,
		}
		if ngmp.MaintenanceWindow != nil {
			res.MaintenanceWindow = NewMaintenanceWindow(ngmp.MaintenanceWindow)
		}

		return res
	}
	return nil
}

func NewNodeGroupMaintenancePolicySDK(ngmp *k8s_pb.NodeGroupMaintenancePolicy) *v1alpha1.NodeGroupMaintenancePolicy {
	if ngmp != nil {
		res := &v1alpha1.NodeGroupMaintenancePolicy{
			AutoUpgrade: ngmp.AutoUpgrade,
			AutoRepair:  ngmp.AutoRepair,
		}
		if ngmp.MaintenanceWindow != nil {
			res.MaintenanceWindow = NewMaintenanceWindowSDK(ngmp.MaintenanceWindow)
		}

		return res
	}
	return nil
}

func NewDeployPolicy(dp *v1alpha1.DeployPolicy) *k8s_pb.DeployPolicy {
	if dp != nil {
		return &k8s_pb.DeployPolicy{
			MaxUnavailable: dp.MaxUnavailable,
			MaxExpansion:   dp.MaxExpansion,
		}
	}
	return nil
}

func NewDeployPolicySDK(dp *k8s_pb.DeployPolicy) *v1alpha1.DeployPolicy {
	if dp != nil {
		return &v1alpha1.DeployPolicy{
			MaxUnavailable: dp.MaxUnavailable,
			MaxExpansion:   dp.MaxExpansion,
		}
	}
	return nil
}

func NewNodeGroupLocation(ngl *v1alpha1.NodeGroupLocation) *k8s_pb.NodeGroupLocation {
	if ngl != nil {
		return &k8s_pb.NodeGroupLocation{
			ZoneId:   ngl.ZoneId,
			SubnetId: ngl.SubnetId,
		}
	}
	return nil
}

func NewNodeGroupLocationSDK(ngl *k8s_pb.NodeGroupLocation) *v1alpha1.NodeGroupLocation {
	if ngl != nil {
		return &v1alpha1.NodeGroupLocation{
			ZoneId:   ngl.ZoneId,
			SubnetId: ngl.SubnetId,
		}
	}
	return nil
}

func NewNodeGroupLocations(ngls v1alpha1.NodeGroupLocations) []*k8s_pb.NodeGroupLocation {
	res := []*k8s_pb.NodeGroupLocation{}
	for _, i := range ngls {
		v := NewNodeGroupLocation(i)
		if v != nil {
			res = append(res, v)
		}
	}
	return res
}

func NewNodeGroupLocationsSDK(ngls []*k8s_pb.NodeGroupLocation) v1alpha1.NodeGroupLocations {
	res := v1alpha1.NodeGroupLocations{}
	for _, i := range ngls {
		v := NewNodeGroupLocationSDK(i)
		if v != nil {
			res = append(res, v)
		}
	}
	return res
}

func NewNodeTemplateNetworkSettings(nt *v1alpha1.NodeTemplate_NetworkSettings) *k8s_pb.NodeTemplate_NetworkSettings {
	if nt != nil {
		var t k8s_pb.NodeTemplate_NetworkSettings_Type
		val, ok := k8s_pb.NodeTemplate_NetworkSettings_Type_value[nt.Type]
		if ok {
			t = k8s_pb.NodeTemplate_NetworkSettings_Type(val)
		} else {
			t = k8s_pb.NodeTemplate_NetworkSettings_TYPE_UNSPECIFIED
		}
		return &k8s_pb.NodeTemplate_NetworkSettings{
			Type: t,
		}
	}
	return nil
}

func NewNodeTemplateNetworkSettingsSDK(nt *k8s_pb.NodeTemplate_NetworkSettings) *v1alpha1.NodeTemplate_NetworkSettings {
	if nt != nil {
		return &v1alpha1.NodeTemplate_NetworkSettings{
			Type: nt.Type.String(),
		}
	}
	return nil
}

func NewPlacementPolicy(pp *v1alpha1.PlacementPolicy) *k8s_pb.PlacementPolicy {
	if pp != nil {
		return &k8s_pb.PlacementPolicy{
			PlacementGroupId: pp.PlacementGroupId,
		}
	}
	return nil
}

func NewPlacementPolicySDK(pp *k8s_pb.PlacementPolicy) *v1alpha1.PlacementPolicy {
	if pp != nil {
		return &v1alpha1.PlacementPolicy{
			PlacementGroupId: pp.PlacementGroupId,
		}
	}
	return nil
}

func NewNetworkInterfaceSpec(nis *v1alpha1.NetworkInterfaceSpec) *k8s_pb.NetworkInterfaceSpec {
	if nis != nil {
		r := &k8s_pb.NetworkInterfaceSpec{
			SubnetIds:        nis.SubnetIds,
			SecurityGroupIds: nis.SecurityGroupIds,
		}
		if nis.PrimaryV4AddressSpec != nil {
			var ipv k8s_pb.IpVersion
			val, ok := k8s_pb.IpVersion_value[nis.PrimaryV4AddressSpec.OneToOneNatSpec.IpVersion]
			if ok {
				ipv = k8s_pb.IpVersion(val)
			} else {
				ipv = k8s_pb.IpVersion_IP_VERSION_UNSPECIFIED
			}
			r.PrimaryV4AddressSpec = &k8s_pb.NodeAddressSpec{
				OneToOneNatSpec: &k8s_pb.OneToOneNatSpec{
					IpVersion: ipv,
				},
			}
		}
		if nis.PrimaryV6AddressSpec != nil {
			var ipv k8s_pb.IpVersion
			val, ok := k8s_pb.IpVersion_value[nis.PrimaryV6AddressSpec.OneToOneNatSpec.IpVersion]
			if ok {
				ipv = k8s_pb.IpVersion(val)
			} else {
				ipv = k8s_pb.IpVersion_IP_VERSION_UNSPECIFIED
			}
			r.PrimaryV6AddressSpec = &k8s_pb.NodeAddressSpec{
				OneToOneNatSpec: &k8s_pb.OneToOneNatSpec{
					IpVersion: ipv,
				},
			}
		}
		return r
	}
	return nil
}

func NewNetworkInterfaceSpecSDK(nis *k8s_pb.NetworkInterfaceSpec) *v1alpha1.NetworkInterfaceSpec {
	if nis != nil {
		r := &v1alpha1.NetworkInterfaceSpec{
			SubnetIds:        nis.SubnetIds,
			SecurityGroupIds: nis.SecurityGroupIds,
		}
		if nis.PrimaryV4AddressSpec != nil && nis.PrimaryV4AddressSpec.OneToOneNatSpec != nil {
			ipv := nis.PrimaryV4AddressSpec.OneToOneNatSpec.IpVersion.String()
			r.PrimaryV4AddressSpec = &v1alpha1.NodeAddressSpec{
				OneToOneNatSpec: &v1alpha1.OneToOneNatSpec{
					IpVersion: ipv,
				},
			}
		}
		if nis.PrimaryV6AddressSpec != nil && nis.PrimaryV6AddressSpec.OneToOneNatSpec != nil {
			ipv := nis.PrimaryV6AddressSpec.OneToOneNatSpec.IpVersion.String()
			r.PrimaryV6AddressSpec = &v1alpha1.NodeAddressSpec{
				OneToOneNatSpec: &v1alpha1.OneToOneNatSpec{
					IpVersion: ipv,
				},
			}
		}
		return r
	}
	return nil
}

func NewNetworkInterfaceSpecs(nis v1alpha1.NetworkInterfaceSpecs) []*k8s_pb.NetworkInterfaceSpec {
	res := []*k8s_pb.NetworkInterfaceSpec{}
	for _, n := range nis {
		tp := NewNetworkInterfaceSpec(n)
		if tp != nil {
			res = append(res, tp)
		}
	}
	return res
}

func NewNetworkInterfaceSpecsSDK(nis []*k8s_pb.NetworkInterfaceSpec) v1alpha1.NetworkInterfaceSpecs {
	res := v1alpha1.NetworkInterfaceSpecs{}
	for _, n := range nis {
		tp := NewNetworkInterfaceSpecSDK(n)
		if tp != nil {
			res = append(res, tp)
		}
	}
	return res
}

func NewSchedulingPolicy(sp *v1alpha1.SchedulingPolicy) *k8s_pb.SchedulingPolicy {
	if sp != nil {
		return &k8s_pb.SchedulingPolicy{Preemptible: sp.Preemptible}
	}
	return nil
}

func NewSchedulingPolicySDK(sp *k8s_pb.SchedulingPolicy) *v1alpha1.SchedulingPolicy {
	if sp != nil {
		return &v1alpha1.SchedulingPolicy{Preemptible: sp.Preemptible}
	}
	return nil
}

func NewDiskSpec(bs *v1alpha1.DiskSpec) *k8s_pb.DiskSpec {
	if bs != nil {
		return &k8s_pb.DiskSpec{
			DiskTypeId: bs.DiskTypeId,
			DiskSize:   bs.DiskSize,
		}
	}
	return nil
}

func NewDiskSpecSDK(bs *k8s_pb.DiskSpec) *v1alpha1.DiskSpec {
	if bs != nil {
		return &v1alpha1.DiskSpec{
			DiskTypeId: bs.DiskTypeId,
			DiskSize:   bs.DiskSize,
		}
	}
	return nil
}

func NewResourceSpec(rs *v1alpha1.ResourcesSpec) *k8s_pb.ResourcesSpec {
	if rs != nil {
		return &k8s_pb.ResourcesSpec{
			Memory:       rs.Memory,
			Cores:        rs.Cores,
			CoreFraction: rs.CoreFraction,
			Gpus:         rs.Gpus,
		}
	}
	return nil
}

func NewResourceSpecSDK(rs *k8s_pb.ResourcesSpec) *v1alpha1.ResourcesSpec {
	if rs != nil {
		return &v1alpha1.ResourcesSpec{
			Memory:       rs.Memory,
			Cores:        rs.Cores,
			CoreFraction: rs.CoreFraction,
			Gpus:         rs.Gpus,
		}
	}
	return nil
}

func NewNodeTemplate(nt *v1alpha1.NodeTemplate) *k8s_pb.NodeTemplate {
	if nt != nil {
		res := &k8s_pb.NodeTemplate{
			PlatformId:            nt.PlatformId,
			ResourcesSpec:         NewResourceSpec(nt.ResourcesSpec),
			BootDiskSpec:          NewDiskSpec(nt.BootDiskSpec),
			Metadata:              nt.Metadata,
			SchedulingPolicy:      NewSchedulingPolicy(nt.SchedulingPolicy),
			NetworkInterfaceSpecs: NewNetworkInterfaceSpecs(nt.NetworkInterfaceSpecs),
			PlacementPolicy:       NewPlacementPolicy(nt.PlacementPolicy),
			NetworkSettings:       NewNodeTemplateNetworkSettings(nt.NetworkSettings),
		}

		return res
	}
	return nil
}

func NewNodeTemplateSDK(nt *k8s_pb.NodeTemplate) *v1alpha1.NodeTemplate {
	if nt != nil {
		res := &v1alpha1.NodeTemplate{
			PlatformId:            nt.PlatformId,
			ResourcesSpec:         NewResourceSpecSDK(nt.ResourcesSpec),
			BootDiskSpec:          NewDiskSpecSDK(nt.BootDiskSpec),
			Metadata:              nt.Metadata,
			SchedulingPolicy:      NewSchedulingPolicySDK(nt.SchedulingPolicy),
			NetworkInterfaceSpecs: NewNetworkInterfaceSpecsSDK(nt.NetworkInterfaceSpecs),
			PlacementPolicy:       NewPlacementPolicySDK(nt.PlacementPolicy),
			NetworkSettings:       NewNodeTemplateNetworkSettingsSDK(nt.NetworkSettings),
		}
		return res
	}
	return nil
}

func NewScalePolicy(sp *v1alpha1.ScalePolicy) *k8s_pb.ScalePolicy {
	if sp != nil {
		res := &k8s_pb.ScalePolicy{}

		if sp.ScaleType.FixedScale != nil && sp.ScaleType.AutoScale != nil {
			return nil
		}

		if sp.ScaleType.FixedScale != nil {
			res.ScaleType = &k8s_pb.ScalePolicy_FixedScale_{
				FixedScale: &k8s_pb.ScalePolicy_FixedScale{
					Size: sp.ScaleType.FixedScale.Size,
				},
			}
		} else if sp.ScaleType.AutoScale != nil {
			res.ScaleType = &k8s_pb.ScalePolicy_AutoScale_{
				AutoScale: &k8s_pb.ScalePolicy_AutoScale{
					MinSize:     sp.ScaleType.AutoScale.MinSize,
					MaxSize:     sp.ScaleType.AutoScale.MaxSize,
					InitialSize: sp.ScaleType.AutoScale.InitialSize,
				},
			}
		} else {
			return nil
		}
		return res
	}
	return nil
}

func NewScalePolicySDK(sp *k8s_pb.ScalePolicy) *v1alpha1.ScalePolicy {
	if sp != nil {
		res := &v1alpha1.ScalePolicy{}

		switch sp.GetScaleType().(type) {
		case *k8s_pb.ScalePolicy_FixedScale_:
			res.ScaleType = &v1alpha1.IsScalePolicy_ScaleType{
				FixedScale: &v1alpha1.ScalePolicy_FixedScale{
					Size: sp.GetFixedScale().GetSize(),
				},
			}
		case *k8s_pb.ScalePolicy_AutoScale_:
			res.ScaleType = &v1alpha1.IsScalePolicy_ScaleType{
				AutoScale: &v1alpha1.ScalePolicy_AutoScale{
					MinSize:     sp.GetAutoScale().GetMinSize(),
					MaxSize:     sp.GetAutoScale().GetMaxSize(),
					InitialSize: sp.GetAutoScale().GetInitialSize(),
				},
			}
		}
		return res
	}
	return nil
}

func NewAllocationPolicy(ap *v1alpha1.NodeGroupAllocationPolicy) *k8s_pb.NodeGroupAllocationPolicy {
	if ap != nil {
		res := &k8s_pb.NodeGroupAllocationPolicy{
			Locations: NewNodeGroupLocations(ap.Locations),
		}
		return res
	}
	return nil
}

func NewAllocationPolicySDK(ap *k8s_pb.NodeGroupAllocationPolicy) *v1alpha1.NodeGroupAllocationPolicy {
	if ap != nil {
		res := &v1alpha1.NodeGroupAllocationPolicy{
			Locations: NewNodeGroupLocationsSDK(ap.Locations),
		}
		return res
	}
	return nil
}

func NewVersionInfoSDK(vi *k8s_pb.VersionInfo) *v1alpha1.VersionInfo {
	if vi != nil {
		return &v1alpha1.VersionInfo{
			CurrentVersion:       vi.CurrentVersion,
			NewRevisionAvailable: vi.NewRevisionAvailable,
			NewRevisionSummary:   vi.NewRevisionSummary,
			VersionDeprecated:    vi.VersionDeprecated,
		}
	}
	return nil
}
