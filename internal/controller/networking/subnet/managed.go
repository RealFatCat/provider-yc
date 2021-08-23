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

package subnet

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	vpc_pb "github.com/yandex-cloud/go-genproto/yandex/cloud/vpc/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
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

	"github.com/RealFatCat/provider-yc/apis/network/v1alpha1"
	apisv1alpha1 "github.com/RealFatCat/provider-yc/apis/v1alpha1"
)

const (
	errNotSubnetType = "managed resource is not a SubnetType custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errGetSDK        = "cannot get YC SDK"

	errGetNetwork = "cannot get Network"

	errNewClient    = "cannot create new Service"
	errCreateSubnet = "cannot create subnet"
	errDeleteSubnet = "cannot delete subnet"
	errGetSubnet    = "cannot get subnet"
	errUpdateSubnet = "cannot update subnet"
)

// Setup adds a controller that reconciles SubnetType managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.SubnetTypeGroupKind)

	o := controller.Options{
		RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.SubnetTypeGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			client: mgr.GetClient(),
		}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.Subnet{}).
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
	cr, ok := mg.(*v1alpha1.Subnet)
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
	cl := sdk.VPC().Subnet()
	n := sdk.VPC().Network()
	fmt.Println("Connecting done")
	return &external{subn: cl, net: n}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	subn *vpc.SubnetServiceClient
	net  *vpc.NetworkServiceClient
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Subnet)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSubnetType)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Observing: %+v", cr)
	req := &vpc_pb.ListSubnetsRequest{FolderId: cr.Spec.ForProvider.FolderID, Filter: fmt.Sprintf("name = '%s'", cr.Spec.ForProvider.Name)}
	resp, err := c.subn.List(ctx, req)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetSubnet)
	}
	if len(resp.Subnets) == 0 {
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

	sn := resp.Subnets[0]
	cr.Status.AtProvider.ID = sn.Id
	cr.Status.AtProvider.FolderID = sn.FolderId
	cr.Status.AtProvider.CreatedAt = sn.CreatedAt.String()
	cr.Status.AtProvider.Name = sn.Name
	cr.Status.AtProvider.Description = sn.Description
	cr.Status.AtProvider.Labels = sn.Labels
	cr.Status.AtProvider.NetworkID = sn.NetworkId
	cr.Status.AtProvider.ZoneID = sn.ZoneId
	cr.Status.AtProvider.V4CidrBlocks = sn.V4CidrBlocks
	cr.Status.AtProvider.V6CidrBlocks = sn.V6CidrBlocks
	cr.Status.AtProvider.RouteTableID = sn.RouteTableId
	cr.Status.AtProvider.Range = fmt.Sprintf("%s %s", strings.Join(cr.Status.AtProvider.V4CidrBlocks, " "), strings.Join(cr.Status.AtProvider.V6CidrBlocks, " "))

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
	cr, ok := mg.(*v1alpha1.Subnet)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotSubnetType)
	}

	fmt.Printf("Creating: %+v", cr)
	cr.Status.SetConditions(xpv1.Creating())

	nreq := &vpc_pb.ListNetworksRequest{
		FolderId: cr.Spec.ForProvider.FolderID,
		Filter:   fmt.Sprintf("name = '%s'", cr.Spec.ForProvider.NetworkName),
	}
	nrsp, err := c.net.List(ctx, nreq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetNetwork)
	}
	if len(nrsp.Networks) != 1 {
		return managed.ExternalCreation{}, errors.Wrap(fmt.Errorf("%s not found, or multiple values recived", cr.Spec.ForProvider.NetworkName), errGetNetwork)
	}
	netID := nrsp.Networks[0].Id

	req := &vpc_pb.CreateSubnetRequest{
		// To get the folder ID, use a [yandex.cloud.resourcemanager.v1.FolderService.List] request.
		FolderId: cr.Spec.ForProvider.FolderID,
		// Name of the subnet.
		// The name must be unique within the folder.
		Name: cr.Spec.ForProvider.Name,
		// Description of the network.
		Description: cr.Spec.ForProvider.Description,
		// Resource labels as `` key:value `` pairs.
		Labels: cr.Spec.ForProvider.Labels,
		// ID of the network to create subnet in.
		NetworkId: netID,
		// ID of the availability zone where the subnet resides.
		// To get a list of available zones, use the [yandex.cloud.compute.v1.ZoneService.List] request.
		ZoneId: cr.Spec.ForProvider.ZoneID,
		// CIDR block.
		// The range of internal addresses that are defined for this subnet.
		// This field can be set only at Subnet resource creation time and cannot be changed.
		// For example, 10.0.0.0/22 or 192.168.0.0/24.
		// Minimum subnet size is /28, maximum subnet size is /16.
		V4CidrBlocks: cr.Spec.ForProvider.V4CidrBlocks,
		// ID of route table the subnet is linked to.
		RouteTableId: cr.Spec.ForProvider.RouteTableID,
	}
	if _, err := c.subn.Create(ctx, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateSubnet)
	}

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Subnet)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotSubnetType)
	}
	req := &vpc_pb.GetSubnetRequest{SubnetId: cr.Status.AtProvider.ID}
	resp, err := c.subn.Get(ctx, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetSubnet)
	}
	ureq := &vpc_pb.UpdateSubnetRequest{
		SubnetId:     resp.Id,
		Name:         cr.Spec.ForProvider.Name,
		Description:  cr.Spec.ForProvider.Description,
		Labels:       cr.Spec.ForProvider.Labels,
		RouteTableId: cr.Spec.ForProvider.RouteTableID,
		// No UpdateMask support for now. It seems useless, when we use yaml files to "rule them all".
		// No DhcpOptions support for now. This resource is not described yet in our apis.
	}
	if cr.Status.AtProvider.Name != cr.Spec.ForProvider.Name ||
		!reflect.DeepEqual(cr.Status.AtProvider.Labels, cr.Spec.ForProvider.Labels) ||
		cr.Status.AtProvider.Description != cr.Spec.ForProvider.Description ||
		cr.Status.AtProvider.RouteTableID != cr.Spec.ForProvider.RouteTableID {

		_, err = c.subn.Update(ctx, ureq)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateSubnet)
		}
	}

	cr.Status.AtProvider.Name = cr.Spec.ForProvider.Name
	cr.Status.AtProvider.Labels = cr.Spec.ForProvider.Labels
	cr.Status.AtProvider.Description = cr.Spec.ForProvider.Description
	cr.Status.AtProvider.RouteTableID = cr.Spec.ForProvider.RouteTableID

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Subnet)
	if !ok {
		return errors.New(errNotSubnetType)
	}

	mg.SetConditions(xpv1.Deleting())

	req := &vpc_pb.DeleteSubnetRequest{SubnetId: cr.Status.AtProvider.ID}
	fmt.Printf("Deleting: %+v", cr)
	_, err := c.subn.Delete(ctx, req)
	if err != nil {
		return errors.Wrap(err, errDeleteSubnet)
	}
	return nil
}
