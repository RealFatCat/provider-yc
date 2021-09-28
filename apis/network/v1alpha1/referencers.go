package v1alpha1

import (
	"context"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

func NetworkID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		s, ok := mg.(*Network)
		if !ok {
			return ""
		}
		return s.Status.AtProvider.ID
	}
}

// ResolveReferences of this Subnet
func (mg *Subnet) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)
	rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.NetworkID,
		Reference:    mg.Spec.NetworkIDRef,
		Selector:     mg.Spec.NetworkIDSelector,
		To:           reference.To{Managed: &Network{}, List: &NetworkList{}},
		Extract:      NetworkID(),
	})
	if err != nil {
		return errors.Wrap(err, "Spec.NetworkID")
	}
	mg.Spec.NetworkID = rsp.ResolvedValue
	mg.Spec.NetworkIDRef = rsp.ResolvedReference
	return nil
}
