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

package networking

import (
	"context"
	"fmt"
	"reflect"

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
	errNotNetworkType = "managed resource is not a NetworkType custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
	errGetSDK         = "cannot get YC SDK"

	errNewClient            = "cannot create new Service"
	errCreateVirtualNetwork = "cannot create network"
	errDeleteVirtualNetwork = "cannot delete network"
	errGetNetwork           = "cannot get network"
	errUpdateNetwork        = "cannot update network"
)

// Setup adds a controller that reconciles NetworkType managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.NetworkTypeGroupKind)

	o := controller.Options{
		RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.NetworkTypeGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			client: mgr.GetClient(),
		}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.Network{}).
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
	cr, ok := mg.(*v1alpha1.Network)
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
	cl := sdk.VPC().Network()
	fmt.Println("Connecting done")
	return &external{client: cl}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client *vpc.NetworkServiceClient
}

func setStatus() {

}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Network)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotNetworkType)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Observing: %+v", cr)
	req := &vpc_pb.ListNetworksRequest{FolderId: cr.Spec.ForProvider.FolderID, Filter: fmt.Sprintf("name = '%s'", cr.Spec.ForProvider.Name)}
	resp, err := c.client.List(ctx, req)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetNetwork)
	}
	if len(resp.Networks) == 0 {
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

	net := resp.Networks[0]
	cr.Status.AtProvider.CreatedAt = net.CreatedAt.String()
	cr.Status.AtProvider.ID = net.Id
	cr.Status.AtProvider.FolderID = net.FolderId
	cr.Status.AtProvider.Name = net.Name
	cr.Status.AtProvider.Labels = net.Labels
	cr.Status.AtProvider.Description = net.Description
	cr.Status.AtProvider.DefaultSecurityGroupID = net.DefaultSecurityGroupId

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
	cr, ok := mg.(*v1alpha1.Network)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotNetworkType)
	}

	fmt.Printf("Creating: %+v", cr)
	cr.Status.SetConditions(xpv1.Creating())

	req := &vpc_pb.CreateNetworkRequest{
		// To get the folder ID, use a [yandex.cloud.resourcemanager.v1.FolderService.List] request.
		FolderId: cr.Spec.ForProvider.FolderID,
		// Name of the network.
		// The name must be unique within the folder.
		Name: cr.Spec.ForProvider.Name,
		// Description of the network.
		Description: cr.Spec.ForProvider.Description,
		// Resource labels as `` key:value `` pairs.
		Labels: cr.Spec.ForProvider.Labels,
	}
	if _, err := c.client.Create(ctx, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateVirtualNetwork)
	}

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Network)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotNetworkType)
	}
	req := &vpc_pb.GetNetworkRequest{NetworkId: cr.Status.AtProvider.ID}
	resp, err := c.client.Get(ctx, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetNetwork)
	}
	ureq := &vpc_pb.UpdateNetworkRequest{
		NetworkId:   resp.Id,
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
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateNetwork)
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
	cr, ok := mg.(*v1alpha1.Network)
	if !ok {
		return errors.New(errNotNetworkType)
	}

	mg.SetConditions(xpv1.Deleting())

	req := &vpc_pb.DeleteNetworkRequest{NetworkId: cr.Status.AtProvider.ID}
	fmt.Printf("Deleting: %+v", cr)
	_, err := c.client.Delete(ctx, req)
	if err != nil {
		return errors.Wrap(err, errDeleteVirtualNetwork)
	}
	return nil
}
