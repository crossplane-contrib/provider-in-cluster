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

package operator

import (
	"context"
	"fmt"
	"github.com/crossplane-contrib/provider-in-cluster/pkg/controller/utils"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strings"

	"github.com/pkg/errors"

	runtimev1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/client/clientset/versioned"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane-contrib/provider-in-cluster/apis/operator/v1alpha1"
	"github.com/crossplane-contrib/provider-in-cluster/pkg/client/olm/operator"
)

const (
	errUnexpectedObject                  = "the managed resource is not a Postgres resource"
)

// SetupOperator adds a controller that reconciles Operators.
func SetupOperator(mgr ctrl.Manager, l logging.Logger) error {
	name := managed.ControllerName(v1alpha1.OperatorGroupKind)
	postgresLogger := l.WithValues("controller", name)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1alpha1.Operator{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.OperatorGroupVersionKind),
			managed.WithExternalConnecter(&connector{kube: mgr.GetClient(), newClientFn: operator.NewClient, logger: postgresLogger}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithLogger(postgresLogger),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

type connector struct {
	kube        client.Client
	newClientFn func(kube client.Client, logger logging.Logger, p olm.Interface) operator.Client
	logger      logging.Logger
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	cs, err := olm.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &external{client: c.newClientFn(c.kube, c.logger, cs), kube: c.kube, logger: c.logger}, nil
}

type external struct {
	client operator.Client
	kube   client.Client
	logger logging.Logger
}

func (e *external) Observe(ctx context.Context, mgd resource.Managed) (managed.ExternalObservation, error) {
	op, ok := mgd.(*v1alpha1.Operator)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errUnexpectedObject)
	}

	// set initial default values
	initializeDefaults(op)

	pm, err := e.client.GetPackageManifest(ctx, op)
	if err != nil || pm == nil {
		e.logger.Debug("Unable to find package manifest")
		return managed.ExternalObservation{}, err
	}

	csv := e.client.ParsePackageManifest(op, pm)

	if csv == nil {
		e.logger.Debug("Unable to parse package manifest")
		return managed.ExternalObservation{}, nil
	}

	e.logger.Debug(fmt.Sprintf("Package manifest parsed - current CSV %s", utils.StringValue(csv)))

	exists, updated := e.client.CheckCSV(ctx, *csv, op)

	if !updated {
		return managed.ExternalObservation{ResourceExists: exists}, nil
	}

	op.SetConditions(runtimev1alpha1.Available())

	return managed.ExternalObservation{ResourceExists: exists, ResourceUpToDate: updated}, nil
}

func (e *external) Create(ctx context.Context, mgd resource.Managed) (managed.ExternalCreation, error) {
	op, ok := mgd.(*v1alpha1.Operator)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errUnexpectedObject)
	}

	err := e.client.CreateOperator(ctx, op)

	return managed.ExternalCreation{}, resource.Ignore(kerrors.IsAlreadyExists, err)
}

func (e *external) Update(ctx context.Context, mgd resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mgd.(*v1alpha1.Operator)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errUnexpectedObject)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mgd resource.Managed) error {
	op, ok := mgd.(*v1alpha1.Operator)
	if !ok {
		return errors.New(errUnexpectedObject)
	}

	pm, err := e.client.GetPackageManifest(ctx, op)
	if err == nil {
		csv := e.client.ParsePackageManifest(op, pm)

		if csv != nil {
			err = e.client.DeleteCSV(ctx, *csv, op)
			if err != nil {
				return err
			}
		}
	}

	return e.client.DeleteSubscription(ctx, op)
}


func initializeDefaults(op *v1alpha1.Operator) bool {
	updated := false
	// We need to set the default namespace here for the PV/PVC.
	if strings.TrimSpace(op.Namespace) == "" {
		op.Namespace = "default"
		updated = true
	}
	return updated
}
