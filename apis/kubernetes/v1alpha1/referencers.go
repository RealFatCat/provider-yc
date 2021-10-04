package v1alpha1

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	network "github.com/RealFatCat/provider-yc/apis/network/v1alpha1"

	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

func SubnetID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		s, ok := mg.(*network.Subnet)
		if !ok {
			return ""
		}
		return s.Status.AtProvider.ID
	}
}

func NetworkID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		s, ok := mg.(*network.Network)
		if !ok {
			return ""
		}
		return s.Status.AtProvider.ID
	}
}

func ClusterID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		s, ok := mg.(*Cluster)
		if !ok {
			return ""
		}
		return s.Status.AtProvider.ID
	}
}

// ResolveReferences of this Subnet
func (mg *Cluster) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)

	if mg.Spec.MasterSpec.MasterType.ZonalMasterSpec != nil && mg.Spec.MasterSpec.MasterType.RegionalMasterSpec != nil {
		return fmt.Errorf("both zonalMasterSpec and regionalMasterSpec are present, shoule be one of")
	} else if mg.Spec.MasterSpec.MasterType.ZonalMasterSpec != nil {
		ipas := mg.Spec.MasterSpec.MasterType.ZonalMasterSpec.InternalV4AddressSpec
		if ipas != nil {
			rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
				CurrentValue: ipas.SubnetID,
				Reference:    ipas.SubnetIDRef,
				Selector:     ipas.SubnetIDSelector,
				To:           reference.To{Managed: &network.Subnet{}, List: &network.SubnetList{}},
				Extract:      SubnetID(),
			})
			if err != nil {
				return errors.Wrap(err, "mg.Spec.MasterSpec.MasterType.ZonalMasterSpec.InternalV4AddressSpec.SpecID")
			}
			ipas.SubnetID = rsp.ResolvedValue
			ipas.SubnetIDRef = rsp.ResolvedReference
		}
	} else if mg.Spec.MasterSpec.MasterType.RegionalMasterSpec != nil {
		for _, l := range mg.Spec.MasterSpec.MasterType.RegionalMasterSpec.Locations {
			ipas := l.InternalV4AddressSpec
			if ipas != nil {
				rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
					CurrentValue: ipas.SubnetID,
					Reference:    ipas.SubnetIDRef,
					Selector:     ipas.SubnetIDSelector,
					To:           reference.To{Managed: &network.Subnet{}, List: &network.SubnetList{}},
					Extract:      SubnetID(),
				})
				if err != nil {
					return errors.Wrap(err, "mg.Spec.MasterSpec.MasterType.RegionalMasterSpec.Locations.SubnetId")
				}
				ipas.SubnetID = rsp.ResolvedValue
				ipas.SubnetIDRef = rsp.ResolvedReference
			}
		}
	}

	rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.NetworkID,
		Reference:    mg.Spec.NetworkIDRef,
		Selector:     mg.Spec.NetworkIDSelector,
		To:           reference.To{Managed: &network.Network{}, List: &network.NetworkList{}},
		Extract:      NetworkID(),
	})
	if err != nil {
		return errors.Wrap(err, "Spec.NetworkID")
	}
	mg.Spec.NetworkID = rsp.ResolvedValue
	mg.Spec.NetworkIDRef = rsp.ResolvedReference

	return nil
}

// ResolveReferences for NodeGroup
func (mg *NodeGroup) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)
	rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ClusterId,
		Reference:    mg.Spec.ClusterIdRef,
		Selector:     mg.Spec.ClusterIdSelector,
		To:           reference.To{Managed: &Cluster{}, List: &ClusterList{}},
		Extract:      ClusterID(),
	})
	if err != nil {
		return errors.Wrap(err, "Spec.ClusterId")
	}
	mg.Spec.ClusterId = rsp.ResolvedValue
	mg.Spec.ClusterIdRef = rsp.ResolvedReference

	if mg.Spec.AllocationPolicy != nil {
		for _, loc := range mg.Spec.AllocationPolicy.Locations {
			rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
				CurrentValue: loc.SubnetId,
				Reference:    loc.SubnetIdRef,
				Selector:     loc.SubnetIdSelector,
				To:           reference.To{Managed: &network.Subnet{}, List: &network.SubnetList{}},
				Extract:      SubnetID(),
			})
			if err != nil {
				return errors.Wrap(err, "Spec.AllocationPolicy.Locations")
			}
			loc.SubnetId = rsp.ResolvedValue
			loc.SubnetIdRef = rsp.ResolvedReference
		}
	}
	// This is an unreliable design :(
	if mg.Spec.NodeTemplate != nil {
		for _, n := range mg.Spec.NodeTemplate.NetworkInterfaceSpecs {
			for i, sn := range n.SubnetIdsRefs {
				rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
					CurrentValue: n.SubnetIds[i],
					Reference:    sn,
					Selector:     n.SubnetIdsSelectors[i],
					To:           reference.To{Managed: &network.Subnet{}, List: &network.SubnetList{}},
					Extract:      SubnetID(),
				})
				if err != nil {
					return errors.Wrap(err, "Spec.NodeTemplate.NetworkInterfaceSpecs")
				}
				n.SubnetIds[i] = rsp.ResolvedValue
				n.SubnetIdsRefs[i] = rsp.ResolvedReference
			}
		}
	}
	return nil
}
