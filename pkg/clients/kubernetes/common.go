package kubernetes

import (
	"github.com/RealFatCat/provider-yc/apis/kubernetes/v1alpha1"
	k8s_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"

	duration "github.com/golang/protobuf/ptypes/duration"
	dayofweek "google.golang.org/genproto/googleapis/type/dayofweek"
	timeofday "google.golang.org/genproto/googleapis/type/timeofday"
)

func NewMaintenanceWindow(mw *v1alpha1.MaintenanceWindow) *k8s_pb.MaintenanceWindow {
	if mw == nil {
		return nil
	}
	if mw.Policy == nil {
		return nil
	}
	res := &k8s_pb.MaintenanceWindow{}

	if mw.Policy.Anytime != nil {
		res.Policy = &k8s_pb.MaintenanceWindow_Anytime{}
	} else if mw.Policy.DailyMaintenanceWindow != nil {
		res.Policy = &k8s_pb.MaintenanceWindow_DailyMaintenanceWindow{
			DailyMaintenanceWindow: &k8s_pb.DailyMaintenanceWindow{
				StartTime: &timeofday.TimeOfDay{
					Hours:   mw.Policy.DailyMaintenanceWindow.StartTime.Hours,
					Minutes: mw.Policy.DailyMaintenanceWindow.StartTime.Minutes,
					Seconds: mw.Policy.DailyMaintenanceWindow.StartTime.Seconds,
					Nanos:   mw.Policy.DailyMaintenanceWindow.StartTime.Nanos,
				},
				Duration: &duration.Duration{
					Seconds: mw.Policy.DailyMaintenanceWindow.Duration.Seconds,
					Nanos:   mw.Policy.DailyMaintenanceWindow.Duration.Nanos,
				},
			},
		}
	} else if mw.Policy.WeeklyMaintenanceWindow != nil {
		pbDaysOfWeek := []*k8s_pb.DaysOfWeekMaintenanceWindow{}
		for _, days := range mw.Policy.WeeklyMaintenanceWindow.DaysOfWeek {
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

		res.Policy = &k8s_pb.MaintenanceWindow_WeeklyMaintenanceWindow{
			WeeklyMaintenanceWindow: &k8s_pb.WeeklyMaintenanceWindow{
				DaysOfWeek: pbDaysOfWeek,
			},
		}
	}
	return res
}

func NewMaintenanceWindowSDK(mw *k8s_pb.MaintenanceWindow) *v1alpha1.MaintenanceWindow {
	if mw == nil {
		return nil
	}
	if mw.Policy == nil {
		return nil
	}

	res := &v1alpha1.MaintenanceWindow{}

	switch mw.GetPolicy().(type) {
	case *k8s_pb.MaintenanceWindow_Anytime:
		res.Policy = &v1alpha1.MaintenanceWindow_Policy{Anytime: &v1alpha1.AnytimeMaintenanceWindow{}}

	case *k8s_pb.MaintenanceWindow_DailyMaintenanceWindow:
		res.Policy = &v1alpha1.MaintenanceWindow_Policy{
			DailyMaintenanceWindow: &v1alpha1.DailyMaintenanceWindow{
				StartTime: &v1alpha1.TimeOfDay{
					Hours:   mw.GetDailyMaintenanceWindow().StartTime.Hours,
					Minutes: mw.GetDailyMaintenanceWindow().StartTime.Minutes,
					Seconds: mw.GetDailyMaintenanceWindow().StartTime.Seconds,
					Nanos:   mw.GetDailyMaintenanceWindow().StartTime.Nanos,
				},
				Duration: &v1alpha1.Duration{
					Seconds: mw.GetDailyMaintenanceWindow().Duration.Seconds,
					Nanos:   mw.GetDailyMaintenanceWindow().Duration.Nanos,
				},
			},
		}
	case *k8s_pb.MaintenanceWindow_WeeklyMaintenanceWindow:
		daysOfWeek := []*v1alpha1.DaysOfWeekMaintenanceWindow{}
		pbdow := mw.GetWeeklyMaintenanceWindow().DaysOfWeek
		for _, dayOfWeek := range pbdow {
			dow := &v1alpha1.DaysOfWeekMaintenanceWindow{
				StartTime: &v1alpha1.TimeOfDay{
					Hours:   dayOfWeek.StartTime.Hours,
					Minutes: dayOfWeek.StartTime.Minutes,
					Seconds: dayOfWeek.StartTime.Seconds,
					Nanos:   dayOfWeek.StartTime.Nanos,
				},
				Duration: &v1alpha1.Duration{
					Seconds: dayOfWeek.Duration.Seconds,
					Nanos:   dayOfWeek.Duration.Nanos,
				},
			}
			days := []string{}
			for _, day := range dayOfWeek.Days {
				days = append(days, dayofweek.DayOfWeek_name[int32(day)])
			}
			dow.Days = days
			daysOfWeek = append(daysOfWeek, dow)
		}

		res.Policy = &v1alpha1.MaintenanceWindow_Policy{
			WeeklyMaintenanceWindow: &v1alpha1.WeeklyMaintenanceWindow{
				DaysOfWeek: daysOfWeek,
			},
		}
	}
	return res
}
