package v1alpha1

import (
	"context"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	network "github.com/RealFatCat/provider-yc/apis/network/v1alpha1"

	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

func SubnetID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		s, ok := mg.(*network.Subnet)
		if !ok {
			return ""
		}
		return s.Status.AtProvider.ID
	}
}

// ResolveReferences of this Subnet
func (mg *Instance) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)
	for _, i := range mg.Spec.NetworkInterfaces {
		rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
			CurrentValue: i.SubnetID,
			Reference:    i.SubnetIDRef,
			Selector:     i.SubnetIDSelector,
			To:           reference.To{Managed: &network.Subnet{}, List: &network.SubnetList{}},
			Extract:      SubnetID(),
		})
		if err != nil {
			return errors.Wrap(err, "Spec.NetworkInterfaces")
		}
		i.SubnetID = rsp.ResolvedValue
		i.SubnetIDRef = rsp.ResolvedReference
	}
	return nil
}
