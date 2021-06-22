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

package clients

import (
	"context"
	"fmt"

	runtimev1alpha1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane-contrib/provider-in-cluster/apis/v1beta1"
)

const (
	errFailedToGetSecret                 = "failed to get secret from namespace \"%s\""
	errSecretDataIsNil                   = "secret data is nil"
	errProviderConfigNotSet              = "provider config is not set"
	errProviderNotRetrieved              = "provider could not be retrieved"
	errCredSecretNotSet                  = "provider credentials secret is not set"
	errProviderSecretNotRetrieved        = "secret referred in provider could not be retrieved"
	errFailedToCreateRestConfig          = "cannot create new rest config using provider secret"
	errProviderSecretValueForKeyNotFound = "value for key \"%s\" not found in provider credentials secret"
	errFmtUnsupportedCredSource          = "unsupported credentials source %q"
)

// NewRestConfig returns a rest config given a secret with connection information.
func NewRestConfig(kubeconfig []byte) (*rest.Config, error) {
	ac, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kubeconfig")
	}
	return restConfigFromAPIConfig(ac)
}

// NewKubeClient returns a kubernetes client given a secret with connection
// information.
func NewKubeClient(config *rest.Config) (client.Client, error) {
	kc, err := client.New(config, client.Options{})
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Kubernetes client")
	}

	return kc, nil
}

func restConfigFromAPIConfig(c *api.Config) (*rest.Config, error) {
	if c.CurrentContext == "" {
		return nil, errors.New("currentContext not set in kubeconfig")
	}
	ctx := c.Contexts[c.CurrentContext]
	cluster := c.Clusters[ctx.Cluster]
	if cluster == nil {
		return nil, errors.New(fmt.Sprintf("cluster for currentContext (%s) not found", c.CurrentContext))
	}
	user := c.AuthInfos[ctx.AuthInfo]
	if user == nil {
		return nil, errors.New(fmt.Sprintf("auth info for currentContext (%s) not found", c.CurrentContext))
	}
	return &rest.Config{
		Host:            cluster.Server,
		Username:        user.Username,
		Password:        user.Password,
		BearerToken:     user.Token,
		BearerTokenFile: user.TokenFile,
		Impersonate: rest.ImpersonationConfig{
			UserName: user.Impersonate,
			Groups:   user.ImpersonateGroups,
			Extra:    user.ImpersonateUserExtra,
		},
		AuthProvider: user.AuthProvider,
		ExecProvider: user.Exec,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure:   cluster.InsecureSkipTLSVerify,
			ServerName: cluster.TLSServerName,
			CertData:   user.ClientCertificateData,
			KeyData:    user.ClientKeyData,
			CAData:     cluster.CertificateAuthorityData,
		},
	}, nil
}

// GetSecretData extracts arbitrary data from a Kubernetes secret
func GetSecretData(ctx context.Context, kube client.Client, nn types.NamespacedName) (map[string][]byte, error) {
	s := &corev1.Secret{}
	if err := kube.Get(ctx, nn, s); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(errFailedToGetSecret, nn.Namespace))
	}
	if s.Data == nil {
		return nil, errors.New(errSecretDataIsNil)
	}
	return s.Data, nil
}

// GetProviderConfigRC gets the provider config secret, and parses it into a rest.Config
func GetProviderConfigRC(ctx context.Context, cr resource.Managed, kube client.Client) (*rest.Config, error) { //nolint:gocyclo
	p := &v1beta1.ProviderConfig{}

	if cr.GetProviderConfigReference() == nil {
		return nil, errors.New(errProviderConfigNotSet)
	}

	n := types.NamespacedName{Name: cr.GetProviderConfigReference().Name}
	if err := kube.Get(ctx, n, p); err != nil {
		return nil, errors.Wrap(err, errProviderNotRetrieved)
	}

	s := p.Spec.Credentials.Source
	switch s { //nolint:exhaustive
	case runtimev1alpha1.CredentialsSourceInjectedIdentity:
		rc, err := rest.InClusterConfig()
		if err != nil {
			return nil, errors.Wrap(err, errFailedToCreateRestConfig)
		}
		return rc, nil
	case runtimev1alpha1.CredentialsSourceSecret:
		ref := p.Spec.Credentials.SecretRef
		if ref == nil {
			return nil, errors.New(errCredSecretNotSet)
		}

		key := types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}
		d, err := GetSecretData(ctx, kube, key)
		if err != nil {
			return nil, errors.Wrap(err, errProviderSecretNotRetrieved)
		}
		kc, f := d[ref.Key]
		if !f {
			return nil, errors.Errorf(errProviderSecretValueForKeyNotFound, ref.Key)
		}
		rc, err := NewRestConfig(kc)
		if err != nil {
			return nil, errors.Wrap(err, errFailedToCreateRestConfig)
		}
		return rc, nil
	default:
		return nil, errors.Errorf(errFmtUnsupportedCredSource, s)
	}
}
