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
	"fmt"
	"strconv"
	"strings"

	runtimev1alpha1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/crossplane-contrib/provider-in-cluster/apis/database/v1alpha1"
	clients "github.com/crossplane-contrib/provider-in-cluster/pkg/client"
	"github.com/crossplane-contrib/provider-in-cluster/pkg/client/database/postgres"
	"github.com/crossplane-contrib/provider-in-cluster/pkg/controller/utils"
)

const (
	errUnexpectedObject    = "the managed resource is not a Postgres resource" //nolint:golint
	errDelete              = "failed to delete the Postgres resource"          //nolint:golint
	errDeploymentMsg       = "failed to get postgres deployment"               //nolint:golint
	errServiceMsg          = "failed to get postgres service"                  //nolint:golint
	errPVCCreateMsg        = "failed to create or update postgres PVC"         //nolint:golint
	errDeployCreateMsg     = "failed to create or update postgres deployment"  //nolint:golint
	errSVCCreateMsg        = "failed to create or update postgres service"     //nolint:golint
	errGeneratePasswordMsg = "failed to generate potential postgres password"  //nolint:golint

	// ResourceCredentialsSecretDatabaseKey is the key for the connection secret database
	ResourceCredentialsSecretDatabaseKey = "database"
)

// SetupPostgres adds a controller that reconciles Postgres instances.
func SetupPostgres(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.PostgresGroupKind)
	postgresLogger := l.WithValues("controller", name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.Postgres{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.PostgresGroupVersionKind),
			managed.WithExternalConnecter(&connector{kube: mgr.GetClient(), newClientFn: postgres.NewRoleClient, logger: postgresLogger}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithLogger(postgresLogger),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

type connector struct {
	kube        client.Client
	newClientFn func(kube client.Client) postgres.Client
	logger      logging.Logger
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Postgres)
	if !ok {
		return nil, errors.New(errUnexpectedObject)
	}

	c.logger.Debug("Connecting")

	rc, err := clients.GetProviderConfigRC(ctx, cr, c.kube)
	if err != nil {
		return nil, err
	}

	cs, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return nil, err
	}

	kube, err := client.New(rc, client.Options{})
	if err != nil {
		return nil, err
	}

	return &external{client: c.newClientFn(kube), kube: kube, logger: c.logger, cs: cs}, nil
}

type external struct {
	client postgres.Client
	kube   client.Client
	cs     kubernetes.Interface
	logger logging.Logger
}

func (e *external) Observe(ctx context.Context, mgd resource.Managed) (managed.ExternalObservation, error) {
	ps, ok := mgd.(*v1alpha1.Postgres)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errUnexpectedObject)
	}

	// set initial default values
	initializeDefaults(ps)

	// check deployment status
	dpl := &appsv1.Deployment{}
	err := e.kube.Get(ctx, types.NamespacedName{Name: ps.Name, Namespace: ps.Namespace}, dpl)
	if err != nil {
		e.logger.Debug(errDeploymentMsg, "err", err)
		return managed.ExternalObservation{}, errors.Wrap(resource.IgnoreNotFound(err), errDeploymentMsg)
	}

	// check if deployment is ready and return connection details
	dplAvailable := false
	for _, s := range dpl.Status.Conditions {
		if s.Type == appsv1.DeploymentAvailable && s.Status == v1.ConditionTrue {
			dplAvailable = true
			break
		}
	}

	// deployment is in progress
	if !dplAvailable {
		e.logger.Debug("deployment currently not available")
		return managed.ExternalObservation{ResourceExists: true}, nil
	}

	svc := &v1.Service{}
	err = e.kube.Get(ctx, types.NamespacedName{Name: ps.Name, Namespace: ps.Namespace}, svc)
	if err != nil {
		e.logger.Debug(errServiceMsg, "err", err)
		return managed.ExternalObservation{ResourceExists: true}, errors.Wrap(err, errServiceMsg)
	}

	ip := svc.Spec.ClusterIP
	e.logger.Debug("postgres service", "ip", fmt.Sprintf("%+v", ip))

	ps.SetConditions(runtimev1alpha1.Available())

	return managed.ExternalObservation{ConnectionDetails: map[string][]byte{
		runtimev1alpha1.ResourceCredentialsSecretEndpointKey: []byte(ip),
	}, ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *external) Create(ctx context.Context, mgd resource.Managed) (managed.ExternalCreation, error) {
	ps, ok := mgd.(*v1alpha1.Postgres)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errUnexpectedObject)
	}
	pvc, err := postgres.MakePVCPostgres(ps)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errPVCCreateMsg)
	}
	if _, err := e.client.CreateOrUpdate(ctx, pvc); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errPVCCreateMsg)
	}
	// deploy credentials secret
	password, err := e.client.ParseInputSecret(ctx, *ps)
	if err != nil || password == "" {
		password, err = e.client.GeneratePassword()
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errGeneratePasswordMsg)
		}
	}

	// deploy deployment
	if _, err := e.client.CreateOrUpdate(ctx, postgres.MakePostgresDeployment(ps, password)); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errDeployCreateMsg)
	}
	// deploy service
	if _, err := e.client.CreateOrUpdate(ctx, postgres.MakeDefaultPostgresService(ps)); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errSVCCreateMsg)
	}

	return managed.ExternalCreation{
		ConnectionDetails: map[string][]byte{
			runtimev1alpha1.ResourceCredentialsSecretUserKey:     []byte(utils.StringValue(ps.Spec.ForProvider.MasterUsername)),
			runtimev1alpha1.ResourceCredentialsSecretPasswordKey: []byte(password),
			runtimev1alpha1.ResourceCredentialsSecretPortKey:     []byte(strconv.Itoa(utils.IntValue(ps.Spec.ForProvider.Port))),
			ResourceCredentialsSecretDatabaseKey:                 []byte(utils.StringValue(ps.Spec.ForProvider.Database)),
		},
	}, nil
}

func (e *external) Update(ctx context.Context, mgd resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mgd.(*v1alpha1.Postgres)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errUnexpectedObject)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mgd resource.Managed) error {
	ps, ok := mgd.(*v1alpha1.Postgres)
	if !ok {
		return errors.New(errUnexpectedObject)
	}
	err := e.client.DeletePostgresService(ctx, ps)
	if err != nil {
		return errors.Wrap(err, errDelete)
	}
	err = e.client.DeletePostgresDeployment(ctx, ps)
	if err != nil {
		return errors.Wrap(err, errDelete)
	}
	return errors.Wrap(e.client.DeletePostgresPVC(ctx, ps), errDelete)
}

func initializeDefaults(pg *v1alpha1.Postgres) bool {
	updated := false
	// We need to set the default namespace here for the PV/PVC.
	if strings.TrimSpace(pg.Namespace) == "" {
		pg.Namespace = "default"
	}
	if pg.Spec.ForProvider.StorageClass == nil {
		pg.Spec.ForProvider.StorageClass = utils.String("Standard")
		updated = true
	}
	if pg.Spec.ForProvider.MasterUsername == nil {
		pg.Spec.ForProvider.MasterUsername = utils.String("postgres")
		updated = true
	}
	if pg.Spec.ForProvider.Database == nil {
		pg.Spec.ForProvider.Database = pg.Spec.ForProvider.MasterUsername
		updated = true
	}
	if pg.Spec.ForProvider.Port == nil {
		pg.Spec.ForProvider.Port = utils.Int(postgres.DefaultPostgresPort)
	}
	return updated
}
