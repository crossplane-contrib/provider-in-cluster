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

package fake

import (
	"context"
	"github.com/crossplane-contrib/provider-in-cluster/apis/database/v1alpha1"
	"github.com/crossplane-contrib/provider-in-cluster/pkg/client/database/postgres"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ postgres.Client = &MockPostgresClient{}

type MockPostgresClient struct {
	MockCreateOrUpdate         func(ctx context.Context, postgres runtime.Object) (controllerutil.OperationResult, error)
	MockParseInputSecret       func(ctx context.Context, postgres v1alpha1.Postgres) (string, error)
	MockDeleteBucketPVC        func(ctx context.Context, postgres *v1alpha1.Postgres) error
	MockDeleteBucketDeployment func(ctx context.Context, postgres *v1alpha1.Postgres) error
	MockDeleteBucketService    func(ctx context.Context, postgres *v1alpha1.Postgres) error
	MockGeneratePassword       func() (string, error)
}

func (c MockPostgresClient) GeneratePassword() (string, error) {
	return c.MockGeneratePassword()
}

func (c MockPostgresClient) ParseInputSecret(ctx context.Context, postgres v1alpha1.Postgres) (string, error) {
	return c.MockParseInputSecret(ctx, postgres)
}

func (c MockPostgresClient) DeleteBucketPVC(ctx context.Context, postgres *v1alpha1.Postgres) error {
	return c.MockDeleteBucketPVC(ctx, postgres)
}

func (c MockPostgresClient) DeleteBucketDeployment(ctx context.Context, postgres *v1alpha1.Postgres) error {
	return c.MockDeleteBucketDeployment(ctx, postgres)
}

func (c MockPostgresClient) DeleteBucketService(ctx context.Context, postgres *v1alpha1.Postgres) error {
	return c.MockDeleteBucketService(ctx, postgres)
}

func (c MockPostgresClient) CreateOrUpdate(ctx context.Context, postgres runtime.Object) (controllerutil.OperationResult, error) {
	return c.MockCreateOrUpdate(ctx, postgres)
}
