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

package compute

import (
	"context"
	"fmt"

	compute_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	vpc_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/vpc/v1"

	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/gen/compute"
	"github.com/yandex-cloud/go-sdk/iamkey"

	"github.com/yandex-cloud/go-sdk/gen/vpc"

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

	"github.com/RealFatCat/provider-yc/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/RealFatCat/provider-yc/apis/v1alpha1"
)

const (
	errNotSubnetType = "managed resource is not a SubnetType custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errGetSDK        = "cannot get YC SDK"

	errGetNetwork = "cannot get Network"

	errNewClient      = "cannot create new Service"
	errCreateInstance = "cannot create instance"
	errDeleteInstance = "cannot delete instance"
	errGetInstance    = "cannot get instance"
	errUpdateInstance = "cannot update instance"

	errRequiredField = "missing required field"
)

// Setup adds a controller that reconciles SubnetType managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.ComputeTypeGroupKind)

	o := controller.Options{
		RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ComputeTypeGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			client: mgr.GetClient(),
		}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.Compute{}).
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
	// TODO: Most of this function should be in some AUTH package
	fmt.Println("Connecting")
	cr, ok := mg.(*v1alpha1.Compute)
	if !ok {
		return nil, errors.New(errNotSubnetType)
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
	cl := sdk.Compute().Instance()
	sn := sdk.VPC().Subnet()
	fmt.Println("Connecting done")
	return &external{instance: cl, subnet: sn}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	instance *compute.InstanceServiceClient
	subnet   *vpc.SubnetServiceClient
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Compute)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSubnetType)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Observing: j%+v\n", cr)
	req := &compute_pb.ListInstancesRequest{FolderId: cr.Spec.ForProvider.FolderID, Filter: fmt.Sprintf("name = '%s'", cr.Spec.ForProvider.Name)}

	resp, err := c.instance.List(ctx, req)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetInstance)
	}
	if len(resp.Instances) == 0 {
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

	sn := resp.Instances[0]
	cr.Status.AtProvider.ID = sn.Id
	cr.Status.AtProvider.FolderID = sn.FolderId
	cr.Status.AtProvider.CreatedAt = sn.CreatedAt.String()
	cr.Status.AtProvider.Name = sn.Name
	cr.Status.AtProvider.Description = sn.Description
	cr.Status.AtProvider.Labels = sn.Labels
	cr.Status.AtProvider.ZoneID = sn.ZoneId

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

func getSubnetIDs(ctx context.Context, client *vpc.SubnetServiceClient, folderID string, netIfaces []*v1alpha1.NetworkInterfaceSpec) error {
	for _, iface := range netIfaces {
		req := &vpc_pb.ListSubnetsRequest{
			FolderId: folderID,
			Filter:   fmt.Sprintf("name = '%s'", iface.SubnetName),
		}
		nrsp, err := client.List(ctx, req)
		if err != nil {
			return errors.Wrap(err, errGetNetwork)
		}
		if len(nrsp.Subnets) != 1 {
			return errors.Wrap(fmt.Errorf("%s not found, or multiple values recived", iface.SubnetName), errGetNetwork)
		}
		iface.SubnetID = nrsp.Subnets[0].Id
	}
	return nil
}

func fillDNSRecords(recs []*v1alpha1.DnsRecord) []*compute_pb.DnsRecordSpec {
	res := []*compute_pb.DnsRecordSpec{}
	for _, record := range recs {
		r := &compute_pb.DnsRecordSpec{
			Fqdn:      record.Fqdn,
			DnsZoneId: record.DnsZoneId,
			Ttl:       record.Ttl,
			Ptr:       record.Ptr,
		}
		res = append(res, r)
	}
	return res
}

func formDiskPb(diskSpec *v1alpha1.AttachedDiskSpec) (*compute_pb.AttachedDiskSpec, error) {
	// ID or Spec must be present
	// So, if both are present, return error
	if diskSpec.Disk.ID != "" && diskSpec.Disk.Name != "" {
		return nil, errors.Wrap(fmt.Errorf("disk id OR full disk spec must be present, got both"), errRequiredField)
	}

	bds := &compute_pb.AttachedDiskSpec{
		DeviceName: diskSpec.DeviceName,
		AutoDelete: diskSpec.AutoDelete,
	}
	// TODO: do better
	mode, ok := compute_pb.AttachedDiskSpec_Mode_value[diskSpec.Mode]
	if !ok {
		bds.Mode = compute_pb.AttachedDiskSpec_MODE_UNSPECIFIED
	} else {
		bds.Mode = compute_pb.AttachedDiskSpec_Mode(mode)
	}

	if diskSpec.Disk.ID != "" {
		bds.Disk = &compute_pb.AttachedDiskSpec_DiskId{
			DiskId: diskSpec.Disk.ID,
		}
	} else {
		bdsspk := &compute_pb.AttachedDiskSpec_DiskSpec{
			Name:        diskSpec.Disk.Name,
			Description: diskSpec.Disk.Description,
			TypeId:      diskSpec.Disk.TypeId,
			Size:        diskSpec.Disk.Size,
			BlockSize:   diskSpec.Disk.BlockSize,
		}

		if diskSpec.Disk.DiskPlacementPolicy != nil {
			bdsspk.DiskPlacementPolicy = &compute_pb.DiskPlacementPolicy{
				PlacementGroupId: diskSpec.Disk.DiskPlacementPolicy.PlacementGroupID,
			}
		}
		// Types that are assignable to Source:
		//  *AttachedDiskSpec_DiskSpec_ImageId
		//  *AttachedDiskSpec_DiskSpec_SnapshotId
		if diskSpec.Disk.Source == nil {
			return nil, errors.Wrap(fmt.Errorf("source"), errRequiredField)
		}
		if diskSpec.Disk.Source.ImageID != "" {
			bdsspk.Source = &compute_pb.AttachedDiskSpec_DiskSpec_ImageId{
				ImageId: diskSpec.Disk.Source.ImageID,
			}
		} else if diskSpec.Disk.Source.SnapshotID != "" {
			bdsspk.Source = &compute_pb.AttachedDiskSpec_DiskSpec_SnapshotId{
				SnapshotId: diskSpec.Disk.Source.SnapshotID,
			}
		} else {
			return nil, errors.Wrap(fmt.Errorf("'source' must have 'image_id' or 'snapshot_id"), errCreateInstance)
		}
		bds.Disk = &compute_pb.AttachedDiskSpec_DiskSpec_{
			DiskSpec: bdsspk,
		}
	}
	return bds, nil
}

// TODO: validate functions required
// TODO: split this supermassive black hole into functions
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Compute)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotSubnetType)
	}

	fmt.Printf("Creating: %+v", cr)
	cr.Status.SetConditions(xpv1.Creating())

	err := getSubnetIDs(ctx, c.subnet, cr.Spec.ForProvider.FolderID, cr.Spec.ForProvider.NetworkInterfaces)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetNetwork)
	}

	// Fill Resources, required for creation
	if cr.Spec.ForProvider.Resources == nil {
		return managed.ExternalCreation{}, errors.Wrap(fmt.Errorf("resources"), errRequiredField)
	}
	resources := &compute_pb.ResourcesSpec{
		Cores:        cr.Spec.ForProvider.Resources.Cores,
		Memory:       cr.Spec.ForProvider.Resources.Memory,
		CoreFraction: cr.Spec.ForProvider.Resources.CoreFraction,
		Gpus:         cr.Spec.ForProvider.Resources.Gpus,
	}
	req := &compute_pb.CreateInstanceRequest{
		// To get the folder ID, use a [yandex.cloud.resourcemanager.v1.FolderService.List] request.
		FolderId: cr.Spec.ForProvider.FolderID,
		// The name must be unique within the folder.
		Name: cr.Spec.ForProvider.Name,
		// Description of the network.
		Description: cr.Spec.ForProvider.Description,
		// Resource labels as `` key:value `` pairs.
		Labels:     cr.Spec.ForProvider.Labels,
		PlatformId: cr.Spec.ForProvider.PlatformID,
		// ID of the availability zone where the instance resides.
		// To get a list of available zones, use the [yandex.cloud.compute.v1.ZoneService.List] request.
		ZoneId:           cr.Spec.ForProvider.ZoneID,
		ResourcesSpec:    resources,
		Metadata:         cr.Spec.ForProvider.Metadata,
		Hostname:         cr.Spec.ForProvider.Hostname,
		ServiceAccountId: cr.Spec.ForProvider.ServiceAccountID,
	}

	// Fill NetworkSettings
	if cr.Spec.ForProvider.NetworkSettings != nil {
		ns := &compute_pb.NetworkSettings{}
		nst, ok := compute_pb.NetworkSettings_Type_value[cr.Spec.ForProvider.NetworkSettings.Type]
		if ok {
			ns.Type = compute_pb.NetworkSettings_Type(nst)
		} else {
			ns.Type = compute_pb.NetworkSettings_TYPE_UNSPECIFIED
		}
		req.NetworkSettings = ns
	}

	// Fill PlacementPolicy
	if cr.Spec.ForProvider.PlacementPolicy != nil {
		pp := &compute_pb.PlacementPolicy{
			PlacementGroupId: cr.Spec.ForProvider.PlacementPolicy.PlacementGroupID,
		}
		hars := []*compute_pb.PlacementPolicy_HostAffinityRule{}
		for _, rule := range cr.Spec.ForProvider.PlacementPolicy.HostAffinityRules {
			har := &compute_pb.PlacementPolicy_HostAffinityRule{
				Key:    rule.Key,
				Values: rule.Values,
			}
			op, ok := compute_pb.PlacementPolicy_HostAffinityRule_Operator_value[rule.Op]
			if ok {
				har.Op = compute_pb.PlacementPolicy_HostAffinityRule_Operator(op)
			} else {
				har.Op = compute_pb.PlacementPolicy_HostAffinityRule_OPERATOR_UNSPECIFIED
			}
			hars = append(hars, har)
		}
		pp.HostAffinityRules = hars
		req.PlacementPolicy = pp
	}

	// Fill SchedulingPolicy
	if cr.Spec.ForProvider.SchedulingPolicy != nil {
		sp := &compute_pb.SchedulingPolicy{
			Preemptible: cr.Spec.ForProvider.SchedulingPolicy.Preemptible,
		}
		req.SchedulingPolicy = sp
	}

	// Fill NetworkInterfaceSpecs
	if len(cr.Spec.ForProvider.NetworkInterfaces) != 0 {
		nis := []*compute_pb.NetworkInterfaceSpec{}
		for _, iface := range cr.Spec.ForProvider.NetworkInterfaces {

			var pr4a *compute_pb.PrimaryAddressSpec

			if iface.PrimaryV4Address != nil {
				pr4adnsrecords := fillDNSRecords(iface.PrimaryV4Address.DnsRecords)
				pr4a = &compute_pb.PrimaryAddressSpec{
					Address:        iface.PrimaryV4Address.Address,
					DnsRecordSpecs: pr4adnsrecords,
				}

				var otonat *compute_pb.OneToOneNatSpec
				if iface.PrimaryV4Address.OneToOneNatSpec != nil {
					otonat = &compute_pb.OneToOneNatSpec{}
					otodnsrecords := fillDNSRecords(iface.PrimaryV4Address.OneToOneNatSpec.DnsRecords)
					otonat.Address = iface.PrimaryV4Address.OneToOneNatSpec.Address
					otonat.DnsRecordSpecs = otodnsrecords
					// TODO: do better
					ipv, ok := compute_pb.IpVersion_value[iface.PrimaryV4Address.OneToOneNatSpec.IpVersion]
					if ok {
						otonat.IpVersion = compute_pb.IpVersion(ipv)
					} else {
						otonat.IpVersion = compute_pb.IpVersion_IP_VERSION_UNSPECIFIED
					}
				}
				if otonat != nil {
					pr4a.OneToOneNatSpec = otonat
				}
			}

			fmt.Println("pr4dns records filled")
			ni := &compute_pb.NetworkInterfaceSpec{
				SubnetId:             iface.SubnetID,
				PrimaryV4AddressSpec: pr4a,
				// PrimaryV6Address: iface.PrimaryV6Address,
				SecurityGroupIds: iface.SecurityGroupIds,
			}
			nis = append(nis, ni)
		}
		req.NetworkInterfaceSpecs = nis
	}

	// Fill BootDiskSpec, required on creation
	if cr.Spec.ForProvider.BootDiskSpec == nil {
		return managed.ExternalCreation{}, errors.Wrap(fmt.Errorf("boot_disk"), errRequiredField)
	}
	/*
		// ID or Spec must be present
		// So, if both are present, return error
		if cr.Spec.ForProvider.BootDiskSpec.Disk.ID != "" && cr.Spec.ForProvider.BootDiskSpec.Disk.Name != "" {
			return managed.ExternalCreation{}, errors.Wrap(fmt.Errorf("disk id OR full disk spec must be present, got both"), errRequiredField)
		}
		bds := &compute_pb.AttachedDiskSpec{
			DeviceName: cr.Spec.ForProvider.BootDiskSpec.DeviceName,
			AutoDelete: cr.Spec.ForProvider.BootDiskSpec.AutoDelete,
		}
		// TODO: do better
		mode, ok := compute_pb.AttachedDiskSpec_Mode_value[cr.Spec.ForProvider.BootDiskSpec.Mode]
		if !ok {
			bds.Mode = compute_pb.AttachedDiskSpec_MODE_UNSPECIFIED
		} else {
			bds.Mode = compute_pb.AttachedDiskSpec_Mode(mode)
		}

		if cr.Spec.ForProvider.BootDiskSpec.Disk.ID != "" {
			bds.Disk = &compute_pb.AttachedDiskSpec_DiskId{
				DiskId: cr.Spec.ForProvider.BootDiskSpec.Disk.ID,
			}
		} else {
			bdsspk := &compute_pb.AttachedDiskSpec_DiskSpec{
				Name:        cr.Spec.ForProvider.BootDiskSpec.Disk.Name,
				Description: cr.Spec.ForProvider.BootDiskSpec.Disk.Description,
				TypeId:      cr.Spec.ForProvider.BootDiskSpec.Disk.TypeId,
				Size:        cr.Spec.ForProvider.BootDiskSpec.Disk.Size,
				BlockSize:   cr.Spec.ForProvider.BootDiskSpec.Disk.BlockSize,
				// Placement policy configuration.
			}

			if cr.Spec.ForProvider.BootDiskSpec.Disk.DiskPlacementPolicy != nil {
				bdsspk.DiskPlacementPolicy = &compute_pb.DiskPlacementPolicy{
					PlacementGroupId: cr.Spec.ForProvider.BootDiskSpec.Disk.DiskPlacementPolicy.PlacementGroupID,
				}
			}
			// Types that are assignable to Source:
			//  *AttachedDiskSpec_DiskSpec_ImageId
			//  *AttachedDiskSpec_DiskSpec_SnapshotId
			if cr.Spec.ForProvider.BootDiskSpec.Disk.Source == nil {
				return managed.ExternalCreation{}, errors.Wrap(fmt.Errorf("source"), errRequiredField)
			}
			if cr.Spec.ForProvider.BootDiskSpec.Disk.Source.ImageID != "" {
				bdsspk.Source = &compute_pb.AttachedDiskSpec_DiskSpec_ImageId{
					ImageId: cr.Spec.ForProvider.BootDiskSpec.Disk.Source.ImageID,
				}
			} else if cr.Spec.ForProvider.BootDiskSpec.Disk.Source.SnapshotID != "" {
				bdsspk.Source = &compute_pb.AttachedDiskSpec_DiskSpec_SnapshotId{
					SnapshotId: cr.Spec.ForProvider.BootDiskSpec.Disk.Source.SnapshotID,
				}
			} else {
				return managed.ExternalCreation{}, errors.Wrap(fmt.Errorf("'source' must have 'image_id' or 'snapshot_id"), errCreateInstance)
			}
			bds.Disk = &compute_pb.AttachedDiskSpec_DiskSpec_{
				DiskSpec: bdsspk,
			}
		}
	*/
	bds, err := formDiskPb(cr.Spec.ForProvider.BootDiskSpec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateInstance)
	}
	req.BootDiskSpec = bds

	// Fill SecondaryDiskSpecs
	if len(cr.Spec.ForProvider.SecondaryDisks) != 0 {
		sds := []*compute_pb.AttachedDiskSpec{}
		for _, diskSpec := range cr.Spec.ForProvider.SecondaryDisks {
			bds, err := formDiskPb(diskSpec)
			if err != nil {
				return managed.ExternalCreation{}, errors.Wrap(err, errCreateInstance)
			}
			sds = append(sds, bds)
		}
		req.SecondaryDiskSpecs = sds
	}

	fmt.Printf("Request: %#v\n", req)
	if _, err := c.instance.Create(ctx, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateInstance)
	}

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Compute)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotSubnetType)
	}
	fmt.Printf("update %+v", cr)
	// TODO: this fields can be updated

	// ID of the Instance resource to update.
	// To get the instance ID, use a [InstanceService.List] request.
	// InstanceId string `protobuf:"bytes,1,opt,name=instance_id,json=instanceId,proto3" json:"instance_id,omitempty"`
	// // Field mask that specifies which fields of the Instance resource are going to be updated.
	// UpdateMask *field_mask.FieldMask `protobuf:"bytes,2,opt,name=update_mask,json=updateMask,proto3" json:"update_mask,omitempty"`
	// // Name of the instance.
	// Name string `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	// // Description of the instance.
	// Description string `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	// // Resource labels as `key:value` pairs.
	// //
	// // Existing set of `labels` is completely replaced by the provided set.
	// Labels map[string]string `protobuf:"bytes,5,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// // ID of the hardware platform configuration for the instance.
	// // This field affects the available values in [resources_spec] field.
	// //
	// // Platforms allows you to create various types of instances: with a large amount of memory,
	// // with a large number of cores, with a burstable performance.
	// // For more information, see [Platforms](/docs/compute/concepts/vm-platforms).
	// PlatformId string `protobuf:"bytes,6,opt,name=platform_id,json=platformId,proto3" json:"platform_id,omitempty"`
	// // Computing resources of the instance, such as the amount of memory and number of cores.
	// // To get a list of available values, see [Levels of core performance](/docs/compute/concepts/performance-levels).
	// ResourcesSpec *ResourcesSpec `protobuf:"bytes,7,opt,name=resources_spec,json=resourcesSpec,proto3" json:"resources_spec,omitempty"`
	// // The metadata `key:value` pairs that will be assigned to this instance. This includes custom metadata and predefined keys.
	// // The total size of all keys and values must be less than 512 KB.
	// //
	// // Existing set of `metadata` is completely replaced by the provided set.
	// //
	// // Values are free-form strings, and only have meaning as interpreted by the programs which configure the instance.
	// // The values must be 256 KB or less.
	// //
	// // For example, you may use the metadata in order to provide your public SSH key to the instance.
	// // For more information, see [Metadata](/docs/compute/concepts/vm-metadata).
	// Metadata map[string]string `protobuf:"bytes,8,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// // ID of the service account to use for [authentication inside the instance](/docs/compute/operations/vm-connect/auth-inside-vm).
	// // To get the service account ID, use a [yandex.cloud.iam.v1.ServiceAccountService.List] request.
	// ServiceAccountId string `protobuf:"bytes,9,opt,name=service_account_id,json=serviceAccountId,proto3" json:"service_account_id,omitempty"`
	// // Network settings.
	// NetworkSettings *NetworkSettings `protobuf:"bytes,10,opt,name=network_settings,json=networkSettings,proto3" json:"network_settings,omitempty"`
	// // Placement policy configuration.
	// PlacementPolicy *PlacementPolicy `protobuf:"bytes,11,opt,name=placement_policy,json=placementPolicy,proto3" json:"placement_policy,omitempty"`

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Compute)
	if !ok {
		return errors.New(errNotSubnetType)
	}

	mg.SetConditions(xpv1.Deleting())

	req := &compute_pb.DeleteInstanceRequest{InstanceId: cr.Status.AtProvider.ID}
	fmt.Printf("Deleting: %+v", cr)
	_, err := c.instance.Delete(ctx, req)
	if err != nil {
		return errors.Wrap(err, errDeleteInstance)
	}
	return nil
}
