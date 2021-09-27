package cluster

import (
	"context"

	duration "github.com/golang/protobuf/ptypes/duration"
	dayofweek "google.golang.org/genproto/googleapis/type/dayofweek"
	timeofday "google.golang.org/genproto/googleapis/type/timeofday"

	"github.com/RealFatCat/provider-yc/apis/kubernetes/v1alpha1"
	k8s_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
)

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
