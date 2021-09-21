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
	"strings"
	// "reflect"

	acc_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1"
	k8s_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
	vpc_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/vpc/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/gen/iam"
	"github.com/yandex-cloud/go-sdk/gen/kubernetes"
	"github.com/yandex-cloud/go-sdk/gen/vpc"
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

	"github.com/yandex-cloud/go-sdk/sdkresolvers"
)

const (
	errNotClusterType = "managed resource is not a ClusterType custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
	errGetSDK         = "cannot get YC SDK"

	errNewClient     = "cannot create new Service"
	errCreateCluster = "cannot create Cluster"
	errDeleteCluster = "cannot delete Cluster"
	errGetCluster    = "cannot get Cluster"
	errUpdateCluster = "cannot update Cluster"

	errGetSubnet         = "cannot get subnet"
	errGetNetwork        = "cannot get network"
	errGetServiceAcc     = "cannot get service account id"
	errGetNodeServiceAcc = "cannot get node service account id"
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
		return nil, errors.New(errNotClusterType)
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
	subn := sdk.VPC().Subnet()
	net := sdk.VPC().Network()
	accs := sdk.IAM().ServiceAccount()
	fmt.Println("Connecting done")
	return &external{client: cl, subnet: subn, net: net, accs: accs}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client *k8s.ClusterServiceClient
	subnet *vpc.SubnetServiceClient
	net    *vpc.NetworkServiceClient
	accs   *iam.ServiceAccountServiceClient
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotClusterType)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Observing: %+v", cr)
	req := &k8s_pb.ListClustersRequest{
		FolderId: cr.Spec.ForProvider.FolderID,
		Filter:   sdkresolvers.CreateResolverFilter("name", cr.Spec.ForProvider.Name),
	}
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

	cluster := resp.Clusters[0]
	cr.Status.AtProvider.CreatedAt = cluster.CreatedAt.String()
	cr.Status.AtProvider.ID = cluster.Id
	cr.Status.AtProvider.FolderID = cluster.FolderId
	cr.Status.AtProvider.Name = cluster.Name
	cr.Status.AtProvider.Labels = cluster.Labels
	cr.Status.AtProvider.Description = cluster.Description
	cr.Status.AtProvider.Status = cluster.Status.String()
	cr.Status.AtProvider.Health = cluster.Health.String()

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

func fillMasterSpecPb(ctx context.Context, c *external, ms *v1alpha1.MasterSpec, folderID string) (*k8s_pb.MasterSpec, error) {
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
		req := &vpc_pb.ListSubnetsRequest{
			FolderId: folderID,
			Filter:   fmt.Sprintf("name = '%s'", ms.MasterType.ZonalMasterSpec.InternalV4AddressSpec.SubnetName),
		}
		resp, err := c.subnet.List(ctx, req)
		if err != nil {
			return nil, errors.Wrap(err, errGetSubnet)
		}

		pbMasterSpec.MasterType = &k8s_pb.MasterSpec_ZonalMasterSpec{
			ZonalMasterSpec: &k8s_pb.ZonalMasterSpec{
				ZoneId: ms.MasterType.ZonalMasterSpec.ZoneId,
				InternalV4AddressSpec: &k8s_pb.InternalAddressSpec{
					SubnetId: resp.Subnets[0].Id,
				},
				// k8s_pb.ExternalAddressSpec is empty for now
				ExternalV4AddressSpec: &k8s_pb.ExternalAddressSpec{},
			},
		}
	} else if ms.MasterType.RegionalMasterSpec != nil {
		pbLocations := []*k8s_pb.MasterLocation{}
		for _, location := range ms.MasterType.RegionalMasterSpec.Locations {
			req := &vpc_pb.ListSubnetsRequest{
				FolderId: folderID,
				Filter:   fmt.Sprintf("name = '%s'", location.InternalV4AddressSpec.SubnetName),
			}
			resp, err := c.subnet.List(ctx, req)
			if err != nil {
				return nil, errors.Wrap(err, errGetSubnet)
			}
			pbl := &k8s_pb.MasterLocation{
				ZoneId: location.ZoneId,
				InternalV4AddressSpec: &k8s_pb.InternalAddressSpec{
					SubnetId: resp.Subnets[0].Id,
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

	// Maintenance_Policy, probably should be another function
	if ms.MaintenancePolicy != nil {
		pbMasterSpec.MaintenancePolicy = &k8s_pb.MasterMaintenancePolicy{
			AutoUpgrade: ms.MaintenancePolicy.AutoUpgrade,
		}
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
		pbMasterSpec.MaintenancePolicy.MaintenanceWindow = pbMaintenanceWindow
	}
	return pbMasterSpec, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotClusterType)
	}

	fmt.Printf("\n\nCreating: %+v\n\n", cr)

	cr.Status.SetConditions(xpv1.Creating())

	nreq := &vpc_pb.ListNetworksRequest{
		FolderId: cr.Spec.ForProvider.FolderID,
		Filter:   fmt.Sprintf("name = '%s'", cr.Spec.ForProvider.NetworkName),
	}
	nrsp, err := c.net.List(ctx, nreq)
	if err != nil {
		return managed.ExternalCreation{}, errors.New(errGetNetwork)
	}

	serviceAccReq := &acc_pb.ListServiceAccountsRequest{
		FolderId: cr.Spec.ForProvider.FolderID,
		// Filter:   sdkresolvers.CreateResolverFilter("name", cr.Spec.ForProvider.ServiceAccountName),
		Filter: fmt.Sprintf("name='%s'", cr.Spec.ForProvider.ServiceAccountName),
	}
	serviceAccResp, err := c.accs.List(ctx, serviceAccReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.New(errGetServiceAcc)
	}

	fmt.Printf("\n serviceAccResp: %s\n", serviceAccResp)

	nodeAccReq := &acc_pb.ListServiceAccountsRequest{
		FolderId: cr.Spec.ForProvider.FolderID,
		// Filter:   sdkresolvers.CreateResolverFilter("name", cr.Spec.ForProvider.NodeServiceAccountName),
		Filter: fmt.Sprintf("name='%s'", cr.Spec.ForProvider.ServiceAccountName),
	}
	nodeAccResp, err := c.accs.List(ctx, nodeAccReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.New(errGetNodeServiceAcc)
	}
	fmt.Printf("\n nodeAccResp: %s\n", nodeAccResp)

	req := &k8s_pb.CreateClusterRequest{
		FolderId:             cr.Spec.ForProvider.FolderID,
		Name:                 cr.Spec.ForProvider.Name,
		Description:          cr.Spec.ForProvider.Description,
		Labels:               cr.Spec.ForProvider.Labels,
		NetworkId:            nrsp.Networks[0].Id,
		ServiceAccountId:     serviceAccResp.ServiceAccounts[0].Id,
		NodeServiceAccountId: nodeAccResp.ServiceAccounts[0].Id,
	}
	// release channel
	if r, ok := k8s_pb.ReleaseChannel_value[strings.ToUpper(cr.Spec.ForProvider.ReleaseChannel)]; ok {
		req.ReleaseChannel = k8s_pb.ReleaseChannel(r)
	} else {
		req.ReleaseChannel = k8s_pb.ReleaseChannel_RELEASE_CHANNEL_UNSPECIFIED
	}
	// Fill MasterSpec, do better
	pbMasterSpec, err := fillMasterSpecPb(ctx, c, cr.Spec.ForProvider.MasterSpec, cr.Spec.ForProvider.FolderID)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateCluster)
	}
	req.MasterSpec = pbMasterSpec

	// IpAllocationPolicy
	req.IpAllocationPolicy = &k8s_pb.IPAllocationPolicy{}
	if cr.Spec.ForProvider.IpAllocationPolicy != nil {
		req.IpAllocationPolicy.ClusterIpv4CidrBlock = cr.Spec.ForProvider.IpAllocationPolicy.ClusterIpv4CidrBlock
		req.IpAllocationPolicy.NodeIpv4CidrMaskSize = cr.Spec.ForProvider.IpAllocationPolicy.NodeIpv4CidrMaskSize
		req.IpAllocationPolicy.ServiceIpv4CidrBlock = cr.Spec.ForProvider.IpAllocationPolicy.ServiceIpv4CidrBlock
		req.IpAllocationPolicy.ClusterIpv6CidrBlock = cr.Spec.ForProvider.IpAllocationPolicy.ClusterIpv6CidrBlock
		req.IpAllocationPolicy.ServiceIpv6CidrBlock = cr.Spec.ForProvider.IpAllocationPolicy.ServiceIpv6CidrBlock
	}

	// internet gateway
	if cr.Spec.ForProvider.InternetGateway != nil {
		req.InternetGateway = &k8s_pb.CreateClusterRequest_GatewayIpv4Address{
			GatewayIpv4Address: cr.Spec.ForProvider.InternetGateway.GatewayIpv4Address,
		}
	}
	// Network policy
	if cr.Spec.ForProvider.NetworkPolicy != nil {
		var provider k8s_pb.NetworkPolicy_Provider
		val, ok := k8s_pb.NetworkPolicy_Provider_value[cr.Spec.ForProvider.NetworkPolicy.Provider]
		if ok {
			provider = k8s_pb.NetworkPolicy_Provider(val)
		} else {
			provider = k8s_pb.NetworkPolicy_PROVIDER_UNSPECIFIED
		}
		req.NetworkPolicy = &k8s_pb.NetworkPolicy{
			Provider: provider,
		}
	}
	// KMS Provider
	if cr.Spec.ForProvider.KmsProvider != nil {
		req.KmsProvider = &k8s_pb.KMSProvider{
			KeyId: cr.Spec.ForProvider.KmsProvider.KeyId,
		}
	}
	if cr.Spec.ForProvider.NetworkImplementation != nil {
		// for now, there is only one option and it is cilium
		var rm k8s_pb.Cilium_RoutingMode
		val, ok := k8s_pb.Cilium_RoutingMode_value[cr.Spec.ForProvider.NetworkImplementation.Cilium.RoutingMode]
		if ok {
			rm = k8s_pb.Cilium_RoutingMode(val)
		} else {
			rm = k8s_pb.Cilium_ROUTING_MODE_UNSPECIFIED
		}
		req.NetworkImplementation = &k8s_pb.CreateClusterRequest_Cilium{
			Cilium: &k8s_pb.Cilium{
				RoutingMode: rm,
			},
		}
	}

	// End of all fills
	fmt.Printf("\n\n REQUEST: %s\n\n", req)

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
		return managed.ExternalUpdate{}, errors.New(errNotClusterType)
	}
	fmt.Println(cr)
	/*
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
	*/

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return errors.New(errNotClusterType)
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
