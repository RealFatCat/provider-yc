package yc

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ycsdk "github.com/yandex-cloud/go-sdk"

	apisv1alpha1 "github.com/RealFatCat/provider-yc/apis/v1alpha1"
)

const (
	ErrCreateConfig = "could not create conifg for sdk"

	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errParseKeyData  = "could not parse key data"
	errCredsCreation = "could not create creds from key data"
)

// CreateSDK creates yc sdk from config
func CreateSDK(ctx context.Context, cfg *ycsdk.Config) (*ycsdk.SDK, error) {
	sdk, err := ycsdk.Build(ctx, *cfg)
	if err != nil {
		return nil, err
	}
	return sdk, nil
}

// GetCredentials gets auth info for provider-yc
func GetCredentials(ctx context.Context, client client.Client, mg resource.Managed) (ycsdk.Credentials, error) {
	t := resource.NewProviderConfigUsageTracker(client, &apisv1alpha1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := client.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, client, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	key, err := iamkey.ReadFromJSONBytes(data)
	if err != nil {
		return nil, errors.Wrap(err, errParseKeyData)
	}
	creds, err := ycsdk.ServiceAccountKey(key)
	if err != nil {
		return nil, errors.Wrap(err, errCredsCreation)
	}
	return creds, nil
}

// CreateConfig creates config for yc sdk
func CreateConfig(ctx context.Context, client client.Client, mg resource.Managed) (*ycsdk.Config, error) {
	creds, err := GetCredentials(ctx, client, mg)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}
	return &ycsdk.Config{Credentials: creds}, nil
}
