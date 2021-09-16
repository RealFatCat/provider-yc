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

package cluster

import (
	"context"
	"fmt"
	"reflect"

	k8s_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/gen/kubernetes"
	"github.com/yandex-cloud/go-sdk/iamkey"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"

	"github.com/RealFatCat/provider-yc/apis/kubernetes/v1alpha1"
	apisv1alpha1 "github.com/RealFatCat/provider-yc/apis/v1alpha1"

	duration "github.com/golang/protobuf/ptypes/duration"
	dayofweek "google.golang.org/genproto/googleapis/type/dayofweek"
	timeofday "google.golang.org/genproto/googleapis/type/timeofday"
)

const (
	errNotNetworkType = "managed resource is not a NetworkType custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
	errGetSDK         = "cannot get YC SDK"

	errNewClient     = "cannot create new Service"
	errCreateCluster = "cannot create Cluster"
	errDeleteCluster = "cannot delete Cluster"
	errGetCluster    = "cannot get Cluster"
	errUpdateCluster = "cannot update Cluster"
)

// Setup adds a controller that reconciles NetworkType managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.ClusterTypeGroupKind)

	o := controller.Options{
		RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ClusterTypeGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			client: mgr.GetClient(),
		}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.Cluster{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	client client.Client
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	fmt.Println("Connecting")
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return nil, errors.New(errNotNetworkType)
	}

	t := resource.NewProviderConfigUsageTracker(c.client, &apisv1alpha1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.client.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.client, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	key, err := iamkey.ReadFromJSONBytes(data)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse key data")
	}
	creds, err := ycsdk.ServiceAccountKey(key)
	if err != nil {
		return nil, errors.Wrap(err, "could not create creds from key data")
	}
	cfg := ycsdk.Config{Credentials: creds}
	sdk, err := ycsdk.Build(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not create SDK")
	}
	cl := sdk.Kubernetes().Cluster()
	fmt.Println("Connecting done")
	return &external{client: cl}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client *k8s.ClusterServiceClient
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotNetworkType)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Observing: %+v", cr)
	req := &k8s_pb.ListClustersRequest{FolderId: cr.Spec.ForProvider.FolderID, Filter: fmt.Sprintf("name = '%s'", cr.Spec.ForProvider.Name)}
	resp, err := c.client.List(ctx, req)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetCluster)
	}
	if len(resp.Clusters) == 0 {
		return managed.ExternalObservation{
			// Return false when the external resource does not exist. This lets
			// the managed resource reconciler know that it needs to call Create to
			// (re)create the resource, or that it has successfully been deleted.
			ResourceExists: false,

			// Return false when the external resource exists, but it not up to date
			// with the desired managed resource state. This lets the managed
			// resource reconciler know that it needs to call Update.
			// ResourceUpToDate: true,

			// Return any details that may be required to connect to the external
			// resource. These will be stored as the connection secret.
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	net := resp.Clusters[0]
	cr.Status.AtProvider.CreatedAt = net.CreatedAt.String()
	cr.Status.AtProvider.ID = net.Id
	cr.Status.AtProvider.FolderID = net.FolderId
	cr.Status.AtProvider.Name = net.Name
	cr.Status.AtProvider.Labels = net.Labels
	cr.Status.AtProvider.Description = net.Description

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		// ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotNetworkType)
	}

	fmt.Printf("Creating: %+v", cr)
	cr.Status.SetConditions(xpv1.Creating())

	req := &k8s_pb.CreateClusterRequest{
		FolderId:             cr.Spec.ForProvider.FolderID,
		Name:                 cr.Spec.ForProvider.Name,
		Description:          cr.Spec.ForProvider.Description,
		Labels:               cr.Spec.ForProvider.Labels,
		NetworkId:            cr.Spec.ForProvider.NetworkId,
		ServiceAccountId:     cr.Spec.ForProvider.ServiceAccountId,
		NodeServiceAccountId: cr.Spec.ForProvider.NodeServiceAccountId,
	}
	// Fill MasterSpec
	pbMasterSpec := &k8s_pb.MasterSpec{}
	pbMasterSpec.Version = cr.Spec.ForProvider.MasterSpec.Version
	pbMasterSpec.SecurityGroupIds = cr.Spec.ForProvider.MasterSpec.SecurityGroupIds

	if cr.Spec.ForProvider.MasterSpec == nil {
		return managed.ExternalCreation{}, errors.Wrap(fmt.Errorf("master spec is nil"), errCreateCluster)
	}
	if cr.Spec.ForProvider.MasterSpec.MasterType.ZonalMasterSpec != nil && cr.Spec.ForProvider.MasterSpec.MasterType.RegionalMasterSpec != nil {
		return managed.ExternalCreation{}, errors.Wrap(fmt.Errorf("zonal_master_spec or regional_master_spec should be present, got both"), errCreateCluster)
	}

	if cr.Spec.ForProvider.MasterSpec.MasterType.ZonalMasterSpec != nil {
		pbMasterSpec.MasterType = &k8s_pb.MasterSpec_ZonalMasterSpec{
			ZonalMasterSpec: &k8s_pb.ZonalMasterSpec{
				ZoneId: cr.Spec.ForProvider.MasterSpec.MasterType.ZonalMasterSpec.ZoneId,
				InternalV4AddressSpec: &k8s_pb.InternalAddressSpec{
					SubnetId: cr.Spec.ForProvider.MasterSpec.MasterType.ZonalMasterSpec.InternalV4AddressSpec.SubnetId,
				},
				// k8s_pb.ExternalAddressSpec is empty for now
				ExternalV4AddressSpec: &k8s_pb.ExternalAddressSpec{},
			},
		}
	} else if cr.Spec.ForProvider.MasterSpec.MasterType.RegionalMasterSpec != nil {
		pbLocations := []*k8s_pb.MasterLocation{}
		for _, location := range cr.Spec.ForProvider.MasterSpec.MasterType.RegionalMasterSpec.Locations {
			pbl := &k8s_pb.MasterLocation{
				ZoneId: location.ZoneId,
				InternalV4AddressSpec: &k8s_pb.InternalAddressSpec{
					SubnetId: location.InternalV4AddressSpec.SubnetId,
				},
			}
			pbLocations = append(pbLocations, pbl)
		}
		pbMasterSpec.MasterType = &k8s_pb.MasterSpec_RegionalMasterSpec{
			RegionalMasterSpec: &k8s_pb.RegionalMasterSpec{
				RegionId:  cr.Spec.ForProvider.MasterSpec.MasterType.RegionalMasterSpec.RegionId,
				Locations: pbLocations,
				// k8s_pb.ExternalAddressSpec is empty for now
				ExternalV4AddressSpec: &k8s_pb.ExternalAddressSpec{},
			},
		}
	} else {
		return managed.ExternalCreation{}, errors.Wrap(fmt.Errorf("zonal_master_spec or regional_master_spec should be present, got none"), errCreateCluster)
	}

	pbMasterSpec.MaintenancePolicy = &k8s_pb.MasterMaintenancePolicy{
		AutoUpgrade: cr.Spec.ForProvider.MasterSpec.MaintenancePolicy.AutoUpgrade,
	}
	pbMaintenanceWindow := &k8s_pb.MaintenanceWindow{}
	if cr.Spec.ForProvider.MasterSpec.MaintenancePolicy.MaintenanceWindow.Policy.Anytime != nil {
		pbMaintenanceWindow.Policy = &k8s_pb.MaintenanceWindow_Anytime{}
	} else if cr.Spec.ForProvider.MasterSpec.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow != nil {
		pbMaintenanceWindow.Policy = &k8s_pb.MaintenanceWindow_DailyMaintenanceWindow{
			DailyMaintenanceWindow: &k8s_pb.DailyMaintenanceWindow{
				StartTime: &timeofday.TimeOfDay{
					Hours:   cr.Spec.ForProvider.MasterSpec.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.StartTime.Hours,
					Minutes: cr.Spec.ForProvider.MasterSpec.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.StartTime.Minutes,
					Seconds: cr.Spec.ForProvider.MasterSpec.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.StartTime.Seconds,
					Nanos:   cr.Spec.ForProvider.MasterSpec.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.StartTime.Nanos,
				},
				Duration: &duration.Duration{
					Seconds: cr.Spec.ForProvider.MasterSpec.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.Duration.Seconds,
					Nanos:   cr.Spec.ForProvider.MasterSpec.MaintenancePolicy.MaintenanceWindow.Policy.DailyMaintenanceWindow.Duration.Nanos,
				},
			},
		}
	} else if cr.Spec.ForProvider.MasterSpec.MaintenancePolicy.MaintenanceWindow.Policy.WeeklyMaintenanceWindow != nil {
		pbDaysOfWeek := []*k8s_pb.DaysOfWeekMaintenanceWindow{}
		for _, days := range cr.Spec.ForProvider.MasterSpec.MaintenancePolicy.MaintenanceWindow.Policy.WeeklyMaintenanceWindow.DaysOfWeek {
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
	pbMasterSpec.MaintenancePolicy.MaintenanceWindow = pbMaintenanceWindow
	req.MasterSpec = pbMasterSpec

	// End of all fills
	if _, err := c.client.Create(ctx, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateCluster)
	}

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotNetworkType)
	}
	req := &k8s_pb.GetClusterRequest{ClusterId: cr.Status.AtProvider.ID}
	resp, err := c.client.Get(ctx, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetCluster)
	}
	ureq := &k8s_pb.UpdateClusterRequest{
		ClusterId:   resp.Id,
		Name:        cr.Spec.ForProvider.Name,
		Description: cr.Spec.ForProvider.Description,
		Labels:      cr.Spec.ForProvider.Labels,
		// No UpdateMask support for now. It seems useless, when we use yaml files to "rule them all".
	}
	// TODO Do better
	if cr.Status.AtProvider.Name != cr.Spec.ForProvider.Name ||
		!reflect.DeepEqual(cr.Status.AtProvider.Labels, cr.Spec.ForProvider.Labels) ||
		cr.Status.AtProvider.Description != cr.Spec.ForProvider.Description {
		_, err = c.client.Update(ctx, ureq)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateCluster)
		}
	}

	cr.Status.AtProvider.Name = cr.Spec.ForProvider.Name
	cr.Status.AtProvider.Labels = cr.Spec.ForProvider.Labels
	cr.Status.AtProvider.Description = cr.Spec.ForProvider.Description

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return errors.New(errNotNetworkType)
	}

	mg.SetConditions(xpv1.Deleting())

	req := &k8s_pb.DeleteClusterRequest{ClusterId: cr.Status.AtProvider.ID}
	fmt.Printf("Deleting: %+v", cr)
	_, err := c.client.Delete(ctx, req)
	if err != nil {
		return errors.Wrap(err, errDeleteCluster)
	}
	return nil
}
