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

package postgres

import (
	"context"
	"reflect"
	"strconv"
	"testing"


	runtimev1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/crossplane-contrib/provider-in-cluster/apis/database/v1alpha1"
	"github.com/crossplane-contrib/provider-in-cluster/pkg/client/database/postgres"
	"github.com/crossplane-contrib/provider-in-cluster/pkg/client/database/postgres/fake"
	"github.com/crossplane-contrib/provider-in-cluster/pkg/controller/utils"
)

var (
	// an arbitrary managed resource
	unexpectedItem resource.Managed
	errBoom        = errors.New("boom")

	DatabaseSize  = "1Gi"
	PostgresName  = "postgresdb"
	serviceIp     = "0.0.0.0"
	userPass      = "password"
	generatedPass = "123asdf"
	username      = "postgres"
	database      = "postgres"
	defaultPort   = 5432
	sc            = "Standard"
)

type args struct {
	pg   postgres.Client
	kube client.Client
	cr   resource.Managed
}

// PostgresModifier is a function which modifies the Postgres for testing
type PostgresModifier func(postgres *v1alpha1.Postgres)

func withUsername(username *string) PostgresModifier {
	return func(postgres *v1alpha1.Postgres) {
		postgres.Spec.ForProvider.MasterUsername = username
	}
}

func withDatabase(database *string) PostgresModifier {
	return func(postgres *v1alpha1.Postgres) {
		postgres.Spec.ForProvider.Database = database
	}
}

func withDatabaseSize(size string) PostgresModifier {
	return func(postgres *v1alpha1.Postgres) {
		postgres.Spec.ForProvider.DatabaseSize = size
	}
}

func withSC(sc *string) PostgresModifier {
	return func(postgres *v1alpha1.Postgres) {
		postgres.Spec.ForProvider.StorageClass = sc
	}
}

func withPort(port *int) PostgresModifier {
	return func(postgres *v1alpha1.Postgres) {
		postgres.Spec.ForProvider.Port = port
	}
}

func withConditions(conditions ...runtimev1alpha1.Condition) PostgresModifier {
	return func(postgres *v1alpha1.Postgres) {
		postgres.Status.Conditions = conditions
	}
}

// Bucket creates a v1beta1 Bucket for use in testing
func Postgres(m ...PostgresModifier) *v1alpha1.Postgres {
	cr := &v1alpha1.Postgres{
		Spec: v1alpha1.PostgresSpec{
			ForProvider: v1alpha1.PostgresParameters{
				DatabaseSize:   DatabaseSize,
				Database:       utils.String(database),
				StorageClass:   utils.String(sc),
				Port:           utils.Int(defaultPort),
				MasterUsername: utils.String(username),
			},
		},
	}
	for _, f := range m {
		f(cr)
	}
	cr.Namespace = "default"
	meta.SetExternalName(cr, PostgresName)
	return cr
}

func TestObserve(t *testing.T) {

	type want struct {
		cr     resource.Managed
		result managed.ExternalObservation
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		"InValidInput": {
			args: args{
				cr: unexpectedItem,
			},
			want: want{
				cr:  unexpectedItem,
				err: errors.New(errUnexpectedObject),
			},
		},
		"ClientErrorDeployment": {
			args: args{
				kube: &test.MockClient{
					MockGet: func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
						return errBoom
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:  Postgres(),
				err: errors.Wrap(errBoom, errDeploymentMsg),
			},
		},
		"DeploymentNotReady": {
			args: args{
				kube: &test.MockClient{
					MockGet: func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
						return nil
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:     Postgres(),
				err:    nil,
				result: managed.ExternalObservation{ResourceExists: true},
			},
		},
		"ClientErrorService": {
			args: args{
				kube: &test.MockClient{
					MockGet: func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
						switch reflect.TypeOf(obj).String() {
						case "*v1.Deployment":
							dpl := obj.(*appsv1.Deployment)
							dpl.Status.Conditions = []appsv1.DeploymentCondition{
								{
									Type:   appsv1.DeploymentAvailable,
									Status: v1.ConditionTrue,
								},
							}
							return nil
						case "*v1.Service":
							return errBoom
						default:
							return nil
						}
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:     Postgres(),
				result: managed.ExternalObservation{ResourceExists: true},
				err:    errors.Wrap(errBoom, errServiceMsg),
			},
		},
		"ValidInput": {
			args: args{
				kube: &test.MockClient{
					MockGet: func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
						switch reflect.TypeOf(obj).String() {
						case "*v1.Deployment":
							dpl := obj.(*appsv1.Deployment)
							dpl.Status.Conditions = []appsv1.DeploymentCondition{
								{
									Type:   appsv1.DeploymentAvailable,
									Status: v1.ConditionTrue,
								},
							}
							return nil
						case "*v1.Service":
							svc := obj.(*v1.Service)
							svc.Spec.ClusterIP = serviceIp
							return nil
						default:
							return nil
						}
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr: Postgres(withConditions(runtimev1alpha1.Available())),
				result: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: map[string][]byte{
					runtimev1alpha1.ResourceCredentialsSecretEndpointKey: []byte(serviceIp),
				}},
				err: nil,
			},
		},
		"ValidInputLateInit": {
			args: args{
				kube: &test.MockClient{
					MockGet: func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
						switch reflect.TypeOf(obj).String() {
						case "*v1.Deployment":
							dpl := obj.(*appsv1.Deployment)
							dpl.Status.Conditions = []appsv1.DeploymentCondition{
								{
									Type:   appsv1.DeploymentAvailable,
									Status: v1.ConditionTrue,
								},
							}
							return nil
						case "*v1.Service":
							svc := obj.(*v1.Service)
							svc.Spec.ClusterIP = serviceIp
							return nil
						default:
							return nil
						}
					},
				},
				cr: Postgres(withUsername(nil), withDatabase(nil), withPort(nil), withSC(nil)),
			},
			want: want{
				cr: Postgres(withConditions(runtimev1alpha1.Available())),
				result: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: map[string][]byte{
					runtimev1alpha1.ResourceCredentialsSecretEndpointKey: []byte(serviceIp),
				}},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{
				client: tc.pg,
				kube:   tc.kube,
				logger: logging.NewNopLogger(),
			}
			o, err := e.Observe(context.Background(), tc.args.cr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.cr, tc.args.cr, test.EquateConditions()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {

	type want struct {
		cr     resource.Managed
		result managed.ExternalCreation
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		"InValidInput": {
			args: args{
				cr: unexpectedItem,
			},
			want: want{
				cr:  unexpectedItem,
				err: errors.New(errUnexpectedObject),
			},
		},
		"ClientError": {
			args: args{
				pg: &fake.MockPostgresClient{
					MockCreateOrUpdate: func(ctx context.Context, postgres runtime.Object) (controllerutil.OperationResult, error) {
						return controllerutil.OperationResultNone, errBoom
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:  Postgres(),
				err: errors.Wrap(errBoom, errPVCCreateMsg),
			},
		},
		"SizeInvalidError": {
			args: args{
				pg: &fake.MockPostgresClient{
					MockCreateOrUpdate: func(ctx context.Context, postgres runtime.Object) (controllerutil.OperationResult, error) {
						return controllerutil.OperationResultNone, errBoom
					},
				},
				cr: Postgres(withDatabaseSize("1Gb")),
			},
			want: want{
				cr:  Postgres(withDatabaseSize("1Gb")),
				err: errors.Wrap(errors.New("quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'"), errPVCCreateMsg),
			},
		},
		"GeneratePasswordError": {
			args: args{
				pg: &fake.MockPostgresClient{
					MockCreateOrUpdate: func(ctx context.Context, postgres runtime.Object) (controllerutil.OperationResult, error) {
						return controllerutil.OperationResultNone, nil
					},
					MockParseInputSecret: func(ctx context.Context, postgres v1alpha1.Postgres) (string, error) {
						return "", errBoom
					},
					MockGeneratePassword: func() (string, error) {
						return "", errBoom
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:  Postgres(),
				err: errors.Wrap(errBoom, errGeneratePasswordMsg),
			},
		},
		"DeployClientGenerateError": {
			args: args{
				pg: &fake.MockPostgresClient{
					MockCreateOrUpdate: func(ctx context.Context, postgres runtime.Object) (controllerutil.OperationResult, error) {
						switch reflect.TypeOf(postgres).String() {
						case "*v1.Deployment":
							return controllerutil.OperationResultNone, errBoom
						case "*v1.Service":
							return controllerutil.OperationResultNone, nil
						default:
							return controllerutil.OperationResultNone, nil
						}
					},
					MockParseInputSecret: func(ctx context.Context, postgres v1alpha1.Postgres) (string, error) {
						return "", errBoom
					},
					MockGeneratePassword: func() (s string, err error) {
						return generatedPass, nil
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:  Postgres(),
				err: errors.Wrap(errBoom, errDeployCreateMsg),
			},
		},
		"DeployClientNoGenerateError": {
			args: args{
				pg: &fake.MockPostgresClient{
					MockCreateOrUpdate: func(ctx context.Context, postgres runtime.Object) (controllerutil.OperationResult, error) {
						switch reflect.TypeOf(postgres).String() {
						case "*v1.Deployment":
							return controllerutil.OperationResultNone, errBoom
						case "*v1.Service":
							return controllerutil.OperationResultNone, nil
						default:
							return controllerutil.OperationResultNone, nil
						}
					},
					MockParseInputSecret: func(ctx context.Context, postgres v1alpha1.Postgres) (string, error) {
						return userPass, nil
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:  Postgres(),
				err: errors.Wrap(errBoom, errDeployCreateMsg),
			},
		},
		"SVCClientError": {
			args: args{
				pg: &fake.MockPostgresClient{
					MockCreateOrUpdate: func(ctx context.Context, postgres runtime.Object) (controllerutil.OperationResult, error) {
						switch reflect.TypeOf(postgres).String() {
						case "*v1.Deployment":
							return controllerutil.OperationResultNone, nil
						case "*v1.Service":
							return controllerutil.OperationResultNone, errBoom
						default:
							return controllerutil.OperationResultNone, nil
						}
					},
					MockParseInputSecret: func(ctx context.Context, postgres v1alpha1.Postgres) (string, error) {
						return "", errBoom
					},
					MockGeneratePassword: func() (s string, err error) {
						return generatedPass, nil
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:  Postgres(),
				err: errors.Wrap(errBoom, errSVCCreateMsg),
			},
		},
		"ValidInput": {
			args: args{
				pg: &fake.MockPostgresClient{
					MockCreateOrUpdate: func(ctx context.Context, postgres runtime.Object) (controllerutil.OperationResult, error) {
						return controllerutil.OperationResultCreated, nil
					},
					MockParseInputSecret: func(ctx context.Context, postgres v1alpha1.Postgres) (string, error) {
						return userPass, nil
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr: Postgres(),
				result: managed.ExternalCreation{ConnectionDetails: map[string][]byte{
					runtimev1alpha1.ResourceCredentialsSecretUserKey:     []byte(username),
					runtimev1alpha1.ResourceCredentialsSecretPasswordKey: []byte(userPass),
					runtimev1alpha1.ResourceCredentialsSecretPortKey:     []byte(strconv.Itoa(defaultPort)),
					ResourceCredentialsSecretDatabaseKey:                 []byte(database),
				}},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{
				client: tc.pg,
				kube:   tc.kube,
				logger: logging.NewNopLogger(),
			}
			o, err := e.Create(context.Background(), tc.args.cr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.cr, tc.args.cr, test.EquateConditions()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	type want struct {
		cr  resource.Managed
		err error
	}

	cases := map[string]struct {
		args
		want
	}{
		"InValidInput": {
			args: args{
				cr: unexpectedItem,
			},
			want: want{
				cr:  unexpectedItem,
				err: errors.New(errUnexpectedObject),
			},
		},
		"ServiceDeleteError": {
			args: args{
				pg: &fake.MockPostgresClient{
					MockDeleteBucketService: func(ctx context.Context, postgres *v1alpha1.Postgres) error {
						return errBoom
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:  Postgres(),
				err: errors.Wrap(errBoom, errDelete),
			},
		},
		"DeploymentDeleteError": {
			args: args{
				pg: &fake.MockPostgresClient{
					MockDeleteBucketService: func(ctx context.Context, postgres *v1alpha1.Postgres) error {
						return nil
					},
					MockDeleteBucketDeployment: func(ctx context.Context, postgres *v1alpha1.Postgres) error {
						return errBoom
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:  Postgres(),
				err: errors.Wrap(errBoom, errDelete),
			},
		},
		"PVCDeleteError": {
			args: args{
				pg: &fake.MockPostgresClient{
					MockDeleteBucketService: func(ctx context.Context, postgres *v1alpha1.Postgres) error {
						return nil
					},
					MockDeleteBucketDeployment: func(ctx context.Context, postgres *v1alpha1.Postgres) error {
						return nil
					},
					MockDeleteBucketPVC: func(ctx context.Context, postgres *v1alpha1.Postgres) error {
						return errBoom
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:  Postgres(),
				err: errors.Wrap(errBoom, errDelete),
			},
		},
		"ValidInput": {
			args: args{
				pg: &fake.MockPostgresClient{
					MockDeleteBucketService: func(ctx context.Context, postgres *v1alpha1.Postgres) error {
						return nil
					},
					MockDeleteBucketDeployment: func(ctx context.Context, postgres *v1alpha1.Postgres) error {
						return nil
					},
					MockDeleteBucketPVC: func(ctx context.Context, postgres *v1alpha1.Postgres) error {
						return nil
					},
				},
				cr: Postgres(),
			},
			want: want{
				cr:  Postgres(),
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{
				client: tc.pg,
				kube:   tc.kube,
				logger: logging.NewNopLogger(),
			}
			err := e.Delete(context.Background(), tc.args.cr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.cr, tc.args.cr, test.EquateConditions()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}
