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

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/crossplane-contrib/provider-in-cluster/apis/database/v1alpha1"
	"github.com/crossplane-contrib/provider-in-cluster/pkg/client/database/postgres"
)

var _ postgres.Client = &MockPostgresClient{}

// MockPostgresClient is the mock client for the postgres client
type MockPostgresClient struct {
	MockCreateOrUpdate           func(ctx context.Context, postgres runtime.Object) (controllerutil.OperationResult, error)
	MockParseInputSecret         func(ctx context.Context, postgres v1alpha1.Postgres) (string, error)
	MockDeletePostgresPVC        func(ctx context.Context, postgres *v1alpha1.Postgres) error
	MockDeletePostgresDeployment func(ctx context.Context, postgres *v1alpha1.Postgres) error
	MockDeletePostgresService    func(ctx context.Context, postgres *v1alpha1.Postgres) error
	MockGeneratePassword         func() (string, error)
}

// GeneratePassword calls the MockGeneratePassword fake function
func (c MockPostgresClient) GeneratePassword() (string, error) {
	return c.MockGeneratePassword()
}

// ParseInputSecret calls the MockParseInputSecret fake function
func (c MockPostgresClient) ParseInputSecret(ctx context.Context, postgres v1alpha1.Postgres) (string, error) {
	return c.MockParseInputSecret(ctx, postgres)
}

// DeletePostgresPVC calls the MockDeletePostgresPVC fake function
func (c MockPostgresClient) DeletePostgresPVC(ctx context.Context, postgres *v1alpha1.Postgres) error {
	return c.MockDeletePostgresPVC(ctx, postgres)
}

// DeletePostgresDeployment calls the MockDeletePostgresDeployment fake function
func (c MockPostgresClient) DeletePostgresDeployment(ctx context.Context, postgres *v1alpha1.Postgres) error {
	return c.MockDeletePostgresDeployment(ctx, postgres)
}

// DeletePostgresService calls the MockDeletePostgresService fake function
func (c MockPostgresClient) DeletePostgresService(ctx context.Context, postgres *v1alpha1.Postgres) error {
	return c.MockDeletePostgresService(ctx, postgres)
}

// CreateOrUpdate calls the MockCreateOrUpdate fake function
func (c MockPostgresClient) CreateOrUpdate(ctx context.Context, postgres runtime.Object) (controllerutil.OperationResult, error) {
	return c.MockCreateOrUpdate(ctx, postgres)
}
