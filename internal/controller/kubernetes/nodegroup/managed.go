package nodegroup

import (
	"context"
	"fmt"
	"reflect"

	k8s_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"

	yc "github.com/RealFatCat/provider-yc/pkg/clients"

	"github.com/pkg/errors"
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
	kubehlp "github.com/RealFatCat/provider-yc/pkg/clients/kubernetes"

	"github.com/yandex-cloud/go-sdk/sdkresolvers"
	"google.golang.org/genproto/protobuf/field_mask"
)

const (
	errNotNodeGroupType = "managed resource is not a NodeGroupType custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errGetSDK           = "cannot get YC SDK"
	errCreateNodeGroup  = "cannot create node group"

	errNewClient     = "cannot create new Service"
	errCreateCluster = "cannot create Cluster"
	errDeleteCluster = "cannot delete Cluster"
	errGetCluster    = "cannot get Cluster"
	errUpdateCluster = "cannot update Cluster"

	errGetServiceAcc     = "cannot get service account id"
	errGetNodeServiceAcc = "cannot get node service account id"

	errNodeTemplateIsNil      = "nodeTemplate is empty"
	errScalePolicyIsNil       = "scalePolicy is empty"
	errGotBothScalyTypes      = "got both FixedScale and AutoScale, should be one of them"
	errBothScalyTypesNil      = "both FixedScale and AutoScale are nil"
	errAllocationPolicyIsNil  = "AllocationPolicy is nil"
	errDeployPolicyIsNil      = "DeployPolicy is nil"
	errMaintenancePolicyIsNil = "MaintenancePolicy is nil"
)

// Setup adds a controller that reconciles NetworkType managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.NodeGroupTypeGroupKind)

	o := controller.Options{
		RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.NodeGroupTypeGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			client: mgr.GetClient(),
		}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.NodeGroup{}).
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
	cr, ok := mg.(*v1alpha1.NodeGroup)
	if !ok {
		return nil, errors.New(errNotNodeGroupType)
	}
	cfg, err := yc.CreateConfig(ctx, c.client, cr)
	if err != nil {
		return nil, errors.Wrap(err, yc.ErrCreateConfig)
	}
	fmt.Println("Connecting done")
	return &external{cfg: cfg}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	cfg *ycsdk.Config
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.NodeGroup)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotNodeGroupType)
	}
	sdk, err := yc.CreateSDK(ctx, c.cfg)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "could not create SDK")
	}
	defer sdk.Shutdown(ctx)
	client := sdk.Kubernetes().NodeGroup()

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("ObservingNodeGroup: %+v", cr)

	req := &k8s_pb.ListNodeGroupsRequest{
		FolderId: cr.Spec.FolderID,
		Filter:   sdkresolvers.CreateResolverFilter("name", cr.GetName()),
	}
	resp, err := client.List(ctx, req)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetCluster)
	}
	if len(resp.NodeGroups) == 0 {
		fmt.Printf("\n\nNodeGroup don't exists\n\n")
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

	ng := resp.NodeGroups[0]

	cr.Status.AtProvider.CreatedAt = ng.CreatedAt.String()
	cr.Status.AtProvider.ID = ng.Id
	cr.Status.AtProvider.Status = ng.Status.String()
	cr.Status.AtProvider.ClusterID = ng.ClusterId
	cr.Status.AtProvider.Name = ng.Name
	cr.Status.AtProvider.Labels = ng.Labels
	cr.Status.AtProvider.NodeLabels = ng.NodeLabels
	cr.Status.AtProvider.Description = ng.Description
	cr.Status.AtProvider.AllowedUnsafeSysctls = ng.AllowedUnsafeSysctls
	cr.Status.AtProvider.NodeTemplate = kubehlp.NewNodeTemplateSDK(ng.NodeTemplate)
	cr.Status.AtProvider.ScalePolicy = kubehlp.NewScalePolicySDK(ng.ScalePolicy)
	cr.Status.AtProvider.AllocationPolicy = kubehlp.NewAllocationPolicySDK(ng.AllocationPolicy)
	cr.Status.AtProvider.DeployPolicy = kubehlp.NewDeployPolicySDK(ng.DeployPolicy)
	cr.Status.AtProvider.NodeTaints = kubehlp.NewTaintsSDK(ng.NodeTaints)
	cr.Status.AtProvider.VersionInfo = kubehlp.NewVersionInfoSDK(ng.VersionInfo)
	cr.Status.AtProvider.MaintenancePolicy = &v1alpha1.NodeGroupMaintenancePolicy{
		AutoUpgrade:       ng.MaintenancePolicy.AutoUpgrade,
		AutoRepair:        ng.MaintenancePolicy.AutoRepair,
		MaintenanceWindow: kubehlp.NewMaintenanceWindowSDK(ng.MaintenancePolicy.MaintenanceWindow),
	}

	cr.SetConditions(xpv1.Available())

	fmt.Printf("\n\nNodeGroup exists\n\n")
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
	cr, ok := mg.(*v1alpha1.NodeGroup)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotNodeGroupType)
	}

	sdk, err := yc.CreateSDK(ctx, c.cfg)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "could not create SDK")
	}
	defer sdk.Shutdown(ctx)
	client := sdk.Kubernetes().NodeGroup()

	cr.Status.SetConditions(xpv1.Creating())

	req := &k8s_pb.CreateNodeGroupRequest{
		ClusterId:            cr.Spec.ClusterId,
		Name:                 cr.GetName(),
		Description:          cr.Spec.Description,
		Labels:               cr.Spec.Labels,
		Version:              cr.Spec.Version,
		AllowedUnsafeSysctls: cr.Spec.AllowedUnsafeSysctls,
		NodeLabels:           cr.Spec.NodeLabels,
	}
	// NodeTemplate
	if cr.Spec.NodeTemplate == nil {
		return managed.ExternalCreation{}, errors.New(errNodeTemplateIsNil)
	}
	req.NodeTemplate = kubehlp.NewNodeTemplate(cr.Spec.NodeTemplate)

	// ScalePolicy
	if cr.Spec.ScalePolicy == nil {
		return managed.ExternalCreation{}, errors.New(errScalePolicyIsNil)
	}
	req.ScalePolicy = kubehlp.NewScalePolicy(cr.Spec.ScalePolicy)

	// AllocationPolicy
	if cr.Spec.AllocationPolicy == nil {
		return managed.ExternalCreation{}, errors.New(errAllocationPolicyIsNil)
	}
	req.AllocationPolicy = kubehlp.NewAllocationPolicy(cr.Spec.AllocationPolicy)

	// DeployPolicy
	if cr.Spec.DeployPolicy == nil {
		return managed.ExternalCreation{}, errors.New(errDeployPolicyIsNil)
	}
	req.DeployPolicy = kubehlp.NewDeployPolicy(cr.Spec.DeployPolicy)

	if cr.Spec.MaintenancePolicy == nil {
		return managed.ExternalCreation{}, errors.New(errMaintenancePolicyIsNil)
	}
	req.MaintenancePolicy = kubehlp.NewNodeGroupMaintenancePolicy(cr.Spec.MaintenancePolicy)

	if len(cr.Spec.NodeTaints) != 0 {
		req.NodeTaints = kubehlp.NewTaints(cr.Spec.NodeTaints)
	}

	fmt.Printf("\n\nCreating NodeGroup, Request:\n%s\n\n", req)

	// End of all fills
	op, err := sdk.WrapOperation(client.Create(ctx, req))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateNodeGroup)
	}
	err = op.Wait(ctx)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateNodeGroup)
	}
	resp, err := op.Response()
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateNodeGroup)
	}

	fmt.Printf("\n\nOperation Res: %s\n\n", resp)

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.NodeGroup)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotNodeGroupType)
	}
	sdk, err := yc.CreateSDK(ctx, c.cfg)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "could not create SDK")
	}
	defer sdk.Shutdown(ctx)
	client := sdk.Kubernetes().NodeGroup()

	req := &k8s_pb.GetNodeGroupRequest{NodeGroupId: cr.Status.AtProvider.ID}
	resp, err := client.Get(ctx, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetCluster)
	}
	ureq := &k8s_pb.UpdateNodeGroupRequest{
		NodeGroupId: resp.Id,
	}

	// TODO: Move to function
	needUpdate := false
	paths := []string{}
	if resp.Status != k8s_pb.NodeGroup_PROVISIONING {
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

		// if coreFractions set to 0 in spec, yc sets it to 100
		if cr.Spec.NodeTemplate.ResourcesSpec.CoreFraction == 0 {
			cr.Spec.NodeTemplate.ResourcesSpec.CoreFraction = 100
		}
		if cr.Spec.NodeTemplate.SchedulingPolicy == nil {
			cr.Spec.NodeTemplate.SchedulingPolicy = cr.Status.AtProvider.NodeTemplate.SchedulingPolicy
		}
		if cr.Spec.NodeTemplate.PlacementPolicy == nil {
			cr.Spec.NodeTemplate.PlacementPolicy = cr.Status.AtProvider.NodeTemplate.PlacementPolicy
		}
		if cr.Spec.NodeTemplate.NetworkSettings == nil {
			cr.Spec.NodeTemplate.NetworkSettings = cr.Status.AtProvider.NodeTemplate.NetworkSettings
		}
		if len(cr.Spec.NodeTemplate.NetworkInterfaceSpecs) == 0 {
			cr.Spec.NodeTemplate.NetworkInterfaceSpecs = cr.Status.AtProvider.NodeTemplate.NetworkInterfaceSpecs
		}
		if !reflect.DeepEqual(cr.Status.AtProvider.NodeTemplate, cr.Spec.NodeTemplate) {
			ureq.NodeTemplate = kubehlp.NewNodeTemplate(cr.Spec.NodeTemplate)
			needUpdate = true
			paths = append(paths, "node_template")
			fmt.Printf("\n\nAtProvider.NodeTemplate: %+v\n\n", cr.Status.AtProvider.NodeTemplate)
			fmt.Printf("\n\nSpec.NodeTemplate: %+v\n\n", cr.Spec.NodeTemplate)
		}
		if !reflect.DeepEqual(cr.Status.AtProvider.ScalePolicy, cr.Spec.ScalePolicy) {
			ureq.ScalePolicy = kubehlp.NewScalePolicy(cr.Spec.ScalePolicy)
			needUpdate = true
			paths = append(paths, "scale_policy")
		}

		// Trying to fill up Spec nil values with those, that generated at work
		for _, l := range cr.Spec.AllocationPolicy.Locations {
			for _, la := range cr.Status.AtProvider.AllocationPolicy.Locations {
				if la.ZoneId == l.ZoneId {
					if l.SubnetId == "" {
						l.SubnetId = la.SubnetId
					}
				}
			}
		}

		if !reflect.DeepEqual(cr.Status.AtProvider.AllocationPolicy, cr.Spec.AllocationPolicy) {
			ureq.AllocationPolicy = kubehlp.NewAllocationPolicy(cr.Spec.AllocationPolicy)
			needUpdate = true
			paths = append(paths, "allocation_policy")
			fmt.Printf("\n\nAtProvider.AllocationPolicy.Locations:\n")
			for _, l := range cr.Status.AtProvider.AllocationPolicy.Locations {
				fmt.Printf("Location%v\n", l)
			}
			fmt.Printf("\n\nSpec.AllocationPolicy: %+v\n\n", cr.Spec.AllocationPolicy)
			for _, l := range cr.Spec.AllocationPolicy.Locations {
				fmt.Printf("Location%v\n", l)
			}
		}
		if !reflect.DeepEqual(cr.Status.AtProvider.DeployPolicy, cr.Spec.DeployPolicy) {
			ureq.DeployPolicy = kubehlp.NewDeployPolicy(cr.Spec.DeployPolicy)
			needUpdate = true
			paths = append(paths, "deploy_policy")
		}
		if !reflect.DeepEqual(cr.Status.AtProvider.DeployPolicy, cr.Spec.DeployPolicy) {
			ureq.DeployPolicy = kubehlp.NewDeployPolicy(cr.Spec.DeployPolicy)
			needUpdate = true
			paths = append(paths, "deploy_policy")
		}
		if cr.Status.AtProvider.VersionInfo.CurrentVersion != cr.Spec.Version {
			ureq.Version = &k8s_pb.UpdateVersionSpec{
				Specifier: &k8s_pb.UpdateVersionSpec_Version{
					Version: cr.Spec.Version,
				},
			}
			needUpdate = true
			paths = append(paths, "version")
		}
		if !reflect.DeepEqual(cr.Status.AtProvider.MaintenancePolicy, cr.Spec.MaintenancePolicy) {
			ureq.MaintenancePolicy = &k8s_pb.NodeGroupMaintenancePolicy{
				AutoUpgrade:       cr.Spec.MaintenancePolicy.AutoUpgrade,
				AutoRepair:        cr.Spec.MaintenancePolicy.AutoRepair,
				MaintenanceWindow: kubehlp.NewMaintenanceWindow(cr.Spec.MaintenancePolicy.MaintenanceWindow),
			}
			needUpdate = true
			paths = append(paths, "maintenance_policy")
		}
		if !reflect.DeepEqual(cr.Status.AtProvider.AllowedUnsafeSysctls, cr.Spec.AllowedUnsafeSysctls) {
			ureq.AllowedUnsafeSysctls = cr.Spec.AllowedUnsafeSysctls
			needUpdate = true
			paths = append(paths, "allowed_unsafe_sysctls")
		}
		if cr.Spec.NodeTaints == nil {
			cr.Spec.NodeTaints = v1alpha1.Taints{}
		}
		if !reflect.DeepEqual(cr.Status.AtProvider.NodeTaints, cr.Spec.NodeTaints) {
			ureq.NodeTaints = kubehlp.NewTaints(cr.Spec.NodeTaints)
			needUpdate = true
			paths = append(paths, "node_taints")
		}
		if !reflect.DeepEqual(cr.Status.AtProvider.NodeLabels, cr.Spec.NodeLabels) {
			ureq.NodeLabels = cr.Spec.NodeLabels
			needUpdate = true
			paths = append(paths, "node_labels")
		}
	}

	if needUpdate {
		updateMask := &field_mask.FieldMask{Paths: paths}
		ureq.UpdateMask = updateMask

		fmt.Printf("\nPaths: %s\n", ureq.UpdateMask.Paths)
		fmt.Printf("\n NodeGroupUpdateReq: %s\n", ureq)

		_, err := client.Update(ctx, ureq)
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
	cr, ok := mg.(*v1alpha1.NodeGroup)
	if !ok {
		return errors.New(errNotNodeGroupType)
	}

	sdk, err := yc.CreateSDK(ctx, c.cfg)
	if err != nil {
		return errors.Wrap(err, "could not create SDK")
	}
	defer sdk.Shutdown(ctx)
	client := sdk.Kubernetes().NodeGroup()
	mg.SetConditions(xpv1.Deleting())

	req := &k8s_pb.DeleteNodeGroupRequest{NodeGroupId: cr.Status.AtProvider.ID}
	fmt.Printf("Deleting: %+v", cr)
	_, err = client.Delete(ctx, req)
	if err != nil {
		return errors.Wrap(err, errDeleteCluster)
	}
	return nil
}
