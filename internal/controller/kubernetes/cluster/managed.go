package cluster

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	k8s_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/gen/iam"
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

	dayofweek "google.golang.org/genproto/googleapis/type/dayofweek"

	"github.com/yandex-cloud/go-sdk/sdkresolvers"

	"google.golang.org/genproto/protobuf/field_mask"

	hlp "github.com/RealFatCat/provider-yc/pkg/clients/kubernetes/cluster"
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
	accs := sdk.IAM().ServiceAccount()
	fmt.Println("Connecting done")
	return &external{client: cl, accs: accs}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client *k8s.ClusterServiceClient
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
		FolderId: cr.Spec.FolderID,
		Filter:   sdkresolvers.CreateResolverFilter("name", cr.GetName()),
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

	igw := &v1alpha1.Cluster_GatewayIpv4Address{GatewayIpv4Address: cluster.GetGatewayIpv4Address()}

	var np *v1alpha1.NetworkPolicy
	if cluster.NetworkPolicy != nil {
		np = &v1alpha1.NetworkPolicy{
			Provider: cluster.NetworkPolicy.GetProvider().String(),
		}
	} else {
		np = &v1alpha1.NetworkPolicy{}
	}

	ms := &v1alpha1.Master{
		Version:          cluster.Master.Version,
		SecurityGroupIds: cluster.Master.SecurityGroupIds,
	}
	mp := &v1alpha1.MasterMaintenancePolicy{
		AutoUpgrade: cluster.Master.MaintenancePolicy.AutoUpgrade,
		MaintenanceWindow: &v1alpha1.MaintenanceWindow{
			Policy: &v1alpha1.MaintenanceWindow_Policy{},
		},
	}

	switch cluster.Master.MaintenancePolicy.MaintenanceWindow.Policy.(type) {
	case *k8s_pb.MaintenanceWindow_Anytime:
		mp.MaintenanceWindow.Policy = &v1alpha1.MaintenanceWindow_Policy{Anytime: &v1alpha1.AnytimeMaintenanceWindow{}}

	case *k8s_pb.MaintenanceWindow_DailyMaintenanceWindow:
		mp.MaintenanceWindow.Policy = &v1alpha1.MaintenanceWindow_Policy{
			DailyMaintenanceWindow: &v1alpha1.DailyMaintenanceWindow{
				StartTime: &v1alpha1.TimeOfDay{
					Hours:   cluster.Master.MaintenancePolicy.MaintenanceWindow.GetDailyMaintenanceWindow().StartTime.Hours,
					Minutes: cluster.Master.MaintenancePolicy.MaintenanceWindow.GetDailyMaintenanceWindow().StartTime.Minutes,
					Seconds: cluster.Master.MaintenancePolicy.MaintenanceWindow.GetDailyMaintenanceWindow().StartTime.Seconds,
					Nanos:   cluster.Master.MaintenancePolicy.MaintenanceWindow.GetDailyMaintenanceWindow().StartTime.Nanos,
				},
				Duration: &v1alpha1.Duration{
					Seconds: cluster.Master.MaintenancePolicy.MaintenanceWindow.GetDailyMaintenanceWindow().Duration.Seconds,
					Nanos:   cluster.Master.MaintenancePolicy.MaintenanceWindow.GetDailyMaintenanceWindow().Duration.Nanos,
				},
			},
		}
	case *k8s_pb.MaintenanceWindow_WeeklyMaintenanceWindow:
		daysOfWeek := []*v1alpha1.DaysOfWeekMaintenanceWindow{}
		pbdow := cluster.Master.MaintenancePolicy.MaintenanceWindow.GetWeeklyMaintenanceWindow().DaysOfWeek
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

		mp.MaintenanceWindow.Policy = &v1alpha1.MaintenanceWindow_Policy{
			WeeklyMaintenanceWindow: &v1alpha1.WeeklyMaintenanceWindow{
				DaysOfWeek: daysOfWeek,
			},
		}
	}
	ms.MaintenancePolicy = mp

	ms.MasterType = &v1alpha1.Master_MasterType{}
	switch cluster.Master.MasterType.(type) {
	case *k8s_pb.Master_ZonalMaster:
		master := cluster.Master.GetZonalMaster()
		ms.MasterType.ZonalMaster = &v1alpha1.ZonalMaster{
			ZoneId:            master.GetZoneId(),
			InternalV4Address: master.GetInternalV4Address(),
			ExternalV4Address: master.GetExternalV4Address(),
		}
	case *k8s_pb.Master_RegionalMaster:
		master := cluster.Master.GetRegionalMaster()
		ms.MasterType.RegionalMaster = &v1alpha1.RegionalMaster{
			RegionId:          master.GetRegionId(),
			InternalV4Address: master.GetInternalV4Address(),
			ExternalV4Address: master.GetExternalV4Address(),
		}
	}

	cr.Status.AtProvider.CreatedAt = cluster.CreatedAt.String()
	cr.Status.AtProvider.ID = cluster.Id
	cr.Status.AtProvider.FolderID = cluster.FolderId
	cr.Status.AtProvider.Name = cluster.Name
	cr.Status.AtProvider.Labels = cluster.Labels
	cr.Status.AtProvider.Description = cluster.Description
	cr.Status.AtProvider.Status = cluster.Status.String()
	cr.Status.AtProvider.Health = cluster.Health.String()
	cr.Status.AtProvider.ServiceAccountId = cluster.ServiceAccountId
	cr.Status.AtProvider.NodeServiceAccountId = cluster.NodeServiceAccountId
	cr.Status.AtProvider.InternetGateway = igw
	cr.Status.AtProvider.NetworkPolicy = np
	cr.Status.AtProvider.Master = ms

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
		return managed.ExternalCreation{}, errors.New(errNotClusterType)
	}

	cr.Status.SetConditions(xpv1.Creating())

	serviceAccID, err := c.getAccID(ctx, cr.Spec.FolderID, cr.Spec.ServiceAccountName)
	if err != nil {
		return managed.ExternalCreation{}, errors.New(errGetServiceAcc)
	}

	nodeAccID, err := c.getAccID(ctx, cr.Spec.FolderID, cr.Spec.NodeServiceAccountName)
	if err != nil {
		return managed.ExternalCreation{}, errors.New(errGetNodeServiceAcc)
	}

	req := &k8s_pb.CreateClusterRequest{
		FolderId:             cr.Spec.FolderID,
		Name:                 cr.GetName(),
		Description:          cr.Spec.Description,
		Labels:               cr.Spec.Labels,
		NetworkId:            cr.Spec.NetworkID,
		ServiceAccountId:     serviceAccID,
		NodeServiceAccountId: nodeAccID,
	}
	// release channel
	if r, ok := k8s_pb.ReleaseChannel_value[strings.ToUpper(cr.Spec.ReleaseChannel)]; ok {
		req.ReleaseChannel = k8s_pb.ReleaseChannel(r)
	} else {
		req.ReleaseChannel = k8s_pb.ReleaseChannel_RELEASE_CHANNEL_UNSPECIFIED
	}
	// Fill MasterSpec, do better
	pbMasterSpec, err := c.fillMasterSpecPb(ctx, cr.Spec.MasterSpec, cr.Spec.FolderID)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateCluster)
	}
	req.MasterSpec = pbMasterSpec

	// IpAllocationPolicy
	req.IpAllocationPolicy = &k8s_pb.IPAllocationPolicy{}
	if cr.Spec.IpAllocationPolicy != nil {
		req.IpAllocationPolicy.ClusterIpv4CidrBlock = cr.Spec.IpAllocationPolicy.ClusterIpv4CidrBlock
		req.IpAllocationPolicy.NodeIpv4CidrMaskSize = cr.Spec.IpAllocationPolicy.NodeIpv4CidrMaskSize
		req.IpAllocationPolicy.ServiceIpv4CidrBlock = cr.Spec.IpAllocationPolicy.ServiceIpv4CidrBlock
		req.IpAllocationPolicy.ClusterIpv6CidrBlock = cr.Spec.IpAllocationPolicy.ClusterIpv6CidrBlock
		req.IpAllocationPolicy.ServiceIpv6CidrBlock = cr.Spec.IpAllocationPolicy.ServiceIpv6CidrBlock
	}

	// internet gateway
	if cr.Spec.InternetGateway != nil {
		req.InternetGateway = &k8s_pb.CreateClusterRequest_GatewayIpv4Address{
			GatewayIpv4Address: cr.Spec.InternetGateway.GatewayIpv4Address,
		}
	}
	// Network policy
	if cr.Spec.NetworkPolicy != nil {
		var provider k8s_pb.NetworkPolicy_Provider
		val, ok := k8s_pb.NetworkPolicy_Provider_value[cr.Spec.NetworkPolicy.Provider]
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
	if cr.Spec.KmsProvider != nil {
		req.KmsProvider = &k8s_pb.KMSProvider{
			KeyId: cr.Spec.KmsProvider.KeyId,
		}
	}
	if cr.Spec.NetworkImplementation != nil {
		// for now, there is only one option and it is cilium
		var rm k8s_pb.Cilium_RoutingMode
		val, ok := k8s_pb.Cilium_RoutingMode_value[cr.Spec.NetworkImplementation.Cilium.RoutingMode]
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
	fmt.Printf("\n\n%+v\n\n", cr)
	req := &k8s_pb.GetClusterRequest{ClusterId: cr.Status.AtProvider.ID}
	resp, err := c.client.Get(ctx, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetCluster)
	}

	svcAccID, err := c.getAccID(ctx, cr.Spec.FolderID, cr.Spec.ServiceAccountName)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetServiceAcc)
	}
	nodeAccID, err := c.getAccID(ctx, cr.Spec.FolderID, cr.Spec.NodeServiceAccountName)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetNodeServiceAcc)
	}
	ureq := &k8s_pb.UpdateClusterRequest{
		ClusterId: resp.Id,
	}

	needUpdate := false
	paths := []string{}
	if cr.Status.AtProvider.Name != cr.GetName() {
		ureq.Name = cr.GetName()
		needUpdate = true
		paths = append(paths, "name")
	}
	if cr.Status.AtProvider.Description != cr.Spec.Description {
		ureq.Description = cr.Spec.Description
		needUpdate = true
		paths = append(paths, "description")
	}
	if !reflect.DeepEqual(cr.Status.AtProvider.Labels, cr.Spec.Labels) {
		ureq.Labels = cr.Spec.Labels
		needUpdate = true
		paths = append(paths, "labels")
	}
	if cr.Status.AtProvider.ServiceAccountId != svcAccID {
		ureq.ServiceAccountId = svcAccID
		needUpdate = true
		paths = append(paths, "service_account_id")
	}
	if cr.Status.AtProvider.NodeServiceAccountId != nodeAccID {
		ureq.NodeServiceAccountId = nodeAccID
		needUpdate = true
		paths = append(paths, "node_service_account_id")
	}
	if !reflect.DeepEqual(cr.Status.AtProvider.InternetGateway, cr.Spec.InternetGateway) {
		if cr.Spec.InternetGateway != nil {
			ureq.InternetGateway = &k8s_pb.UpdateClusterRequest_GatewayIpv4Address{
				GatewayIpv4Address: cr.Spec.InternetGateway.GatewayIpv4Address,
			}
			needUpdate = true
			paths = append(paths, "internet_gateway")
		}
	}
	if !reflect.DeepEqual(cr.Status.AtProvider.NetworkPolicy, cr.Spec.NetworkPolicy) {
		if cr.Spec.NetworkPolicy != nil {
			var p k8s_pb.NetworkPolicy_Provider
			if v, ok := k8s_pb.NetworkPolicy_Provider_value[cr.Spec.NetworkPolicy.Provider]; ok {
				p = k8s_pb.NetworkPolicy_Provider(v)
			} else {
				p = k8s_pb.NetworkPolicy_PROVIDER_UNSPECIFIED
			}
			ureq.NetworkPolicy = &k8s_pb.NetworkPolicy{
				Provider: p,
			}
			needUpdate = true
			paths = append(paths, "network_policy")
		}
	}
	if !reflect.DeepEqual(cr.Status.AtProvider.Master.MaintenancePolicy, cr.Spec.MasterSpec.MaintenancePolicy) ||
		!reflect.DeepEqual(cr.Status.AtProvider.Master.SecurityGroupIds, cr.Spec.MasterSpec.SecurityGroupIds) ||
		cr.Status.AtProvider.Master.Version != cr.Spec.MasterSpec.Version {

		ureq.MasterSpec = &k8s_pb.MasterUpdateSpec{
			Version: &k8s_pb.UpdateVersionSpec{
				// Support only _Version for now
				Specifier: &k8s_pb.UpdateVersionSpec_Version{
					Version: cr.Spec.MasterSpec.Version,
				},
			},
			SecurityGroupIds:  cr.Spec.MasterSpec.SecurityGroupIds,
			MaintenancePolicy: hlp.FillMaintenancePolicyPb(ctx, cr.Spec.MasterSpec),
		}
		needUpdate = true
		paths = append(paths, "master_spec")
	}
	// TODO Do better
	if needUpdate {
		updateMask := &field_mask.FieldMask{Paths: paths}
		ureq.UpdateMask = updateMask

		fmt.Printf("\nPaths: %s\n", ureq.UpdateMask.Paths)
		fmt.Printf("\nUpdate REQ: %s\n", ureq)

		_, err := c.client.Update(ctx, ureq)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateCluster)
		}
	}

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
