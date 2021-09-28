package cluster

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/yandex-cloud/go-sdk/sdkresolvers"

	"github.com/RealFatCat/provider-yc/apis/kubernetes/v1alpha1"

	acc_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1"
	k8s_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"

	hlp "github.com/RealFatCat/provider-yc/pkg/clients/kubernetes/cluster"
)

func (c *external) fillMasterSpecPb(ctx context.Context, ms *v1alpha1.MasterSpec, folderID string) (*k8s_pb.MasterSpec, error) {
	if ms == nil {
		return nil, errors.Wrap(fmt.Errorf("master spec is nil"), errCreateCluster)
	}
	pbMasterSpec := &k8s_pb.MasterSpec{}
	pbMasterSpec.Version = ms.Version
	pbMasterSpec.SecurityGroupIds = ms.SecurityGroupIds

	if ms.MasterType.ZonalMasterSpec != nil && ms.MasterType.RegionalMasterSpec != nil {
		return nil, errors.Wrap(fmt.Errorf("zonal_master_spec or regional_master_spec should be present, got both"), errCreateCluster)
	}

	if ms.MasterType.ZonalMasterSpec != nil {
		// todo: move to function
		pbMasterSpec.MasterType = &k8s_pb.MasterSpec_ZonalMasterSpec{
			ZonalMasterSpec: &k8s_pb.ZonalMasterSpec{
				ZoneId: ms.MasterType.ZonalMasterSpec.ZoneId,
				InternalV4AddressSpec: &k8s_pb.InternalAddressSpec{
					SubnetId: ms.MasterType.ZonalMasterSpec.InternalV4AddressSpec.SubnetID,
				},
				// k8s_pb.ExternalAddressSpec is empty for now
				ExternalV4AddressSpec: &k8s_pb.ExternalAddressSpec{},
			},
		}
	} else if ms.MasterType.RegionalMasterSpec != nil {
		pbLocations := []*k8s_pb.MasterLocation{}
		for _, location := range ms.MasterType.RegionalMasterSpec.Locations {
			pbl := &k8s_pb.MasterLocation{
				ZoneId: location.ZoneId,
				InternalV4AddressSpec: &k8s_pb.InternalAddressSpec{
					SubnetId: location.InternalV4AddressSpec.SubnetID,
				},
			}
			pbLocations = append(pbLocations, pbl)
		}
		pbMasterSpec.MasterType = &k8s_pb.MasterSpec_RegionalMasterSpec{
			RegionalMasterSpec: &k8s_pb.RegionalMasterSpec{
				RegionId:  ms.MasterType.RegionalMasterSpec.RegionId,
				Locations: pbLocations,
				// k8s_pb.ExternalAddressSpec is empty for now
				ExternalV4AddressSpec: &k8s_pb.ExternalAddressSpec{},
			},
		}
	} else {
		return nil, errors.Wrap(fmt.Errorf("zonal_master_spec or regional_master_spec should be present, got none"), errCreateCluster)
	}
	pbMasterSpec.MaintenancePolicy = hlp.FillMaintenancePolicyPb(ctx, ms)
	return pbMasterSpec, nil
}

// TODO: Move to reference
func (c *external) getAccID(ctx context.Context, folderID, accName string) (string, error) {
	req := &acc_pb.ListServiceAccountsRequest{
		FolderId: folderID,
		Filter:   sdkresolvers.CreateResolverFilter("name", accName),
	}
	resp, err := c.accs.List(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.ServiceAccounts[0].Id, nil
}
