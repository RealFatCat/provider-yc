package cluster

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/yandex-cloud/go-sdk/sdkresolvers"

	"github.com/RealFatCat/provider-yc/apis/kubernetes/v1alpha1"

	acc_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1"
	k8s_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"

	duration "github.com/golang/protobuf/ptypes/duration"
	dayofweek "google.golang.org/genproto/googleapis/type/dayofweek"
	timeofday "google.golang.org/genproto/googleapis/type/timeofday"
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
	pbMasterSpec.MaintenancePolicy = FillMaintenancePolicyPb(ctx, ms)
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

func FillMaintenancePolicyPb(ctx context.Context, ms *v1alpha1.MasterSpec) *k8s_pb.MasterMaintenancePolicy {
	res := &k8s_pb.MasterMaintenancePolicy{}
	if ms.MaintenancePolicy == nil {
		return res
	}
	res.AutoUpgrade = ms.MaintenancePolicy.AutoUpgrade

	pbMaintenanceWindow := &k8s_pb.MaintenanceWindow{}
	if ms.MaintenancePolicy.MaintenanceWindow.Policy.Anytime != nil {
		pbMaintenanceWindow.Policy = &k8s_pb.MaintenanceWindow_Anytime{}
	} else if ms.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow != nil {
		pbMaintenanceWindow.Policy = &k8s_pb.MaintenanceWindow_DailyMaintenanceWindow{
			DailyMaintenanceWindow: &k8s_pb.DailyMaintenanceWindow{
				StartTime: &timeofday.TimeOfDay{
					Hours:   ms.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.StartTime.Hours,
					Minutes: ms.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.StartTime.Minutes,
					Seconds: ms.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.StartTime.Seconds,
					Nanos:   ms.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.StartTime.Nanos,
				},
				Duration: &duration.Duration{
					Seconds: ms.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.Duration.Seconds,
					Nanos:   ms.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.Duration.Nanos,
				},
			},
		}
	} else if ms.MaintenancePolicy.MaintenanceWindow.Policy.WeeklyMaintenanceWindow != nil {
		pbDaysOfWeek := []*k8s_pb.DaysOfWeekMaintenanceWindow{}
		for _, days := range ms.MaintenancePolicy.MaintenanceWindow.Policy.WeeklyMaintenanceWindow.DaysOfWeek {
			pbDays := []dayofweek.DayOfWeek{}
			var pbd dayofweek.DayOfWeek
			for _, d := range days.Days {
				if val, ok := dayofweek.DayOfWeek_value[d]; ok {
					pbd = dayofweek.DayOfWeek(val)
				} else {
					pbd = dayofweek.DayOfWeek_DAY_OF_WEEK_UNSPECIFIED
				}
				pbDays = append(pbDays, pbd)
			}
			pbDayOfWeek := &k8s_pb.DaysOfWeekMaintenanceWindow{
				StartTime: &timeofday.TimeOfDay{
					Hours:   days.StartTime.Hours,
					Minutes: days.StartTime.Minutes,
					Seconds: days.StartTime.Seconds,
					Nanos:   days.StartTime.Nanos,
				},
				Duration: &duration.Duration{
					Seconds: days.Duration.Seconds,
					Nanos:   days.Duration.Nanos,
				},
				Days: pbDays,
			}
			pbDaysOfWeek = append(pbDaysOfWeek, pbDayOfWeek)
		}

		pbMaintenanceWindow.Policy = &k8s_pb.MaintenanceWindow_WeeklyMaintenanceWindow{
			WeeklyMaintenanceWindow: &k8s_pb.WeeklyMaintenanceWindow{
				DaysOfWeek: pbDaysOfWeek,
			},
		}
	}
	res.MaintenanceWindow = pbMaintenanceWindow
	return res
}
