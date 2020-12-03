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
	"github.com/crossplane-contrib/provider-in-cluster/apis/operator/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	operaterv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Client = &operatorClient{}

type Client interface {
	CreateOperator(ctx context.Context, obj *v1alpha1.Operator) error
	GetPackageManifest(ctx context.Context, obj *v1alpha1.Operator) (*operatorsv1.PackageManifest, error)
	ParsePackageManifest(op *v1alpha1.Operator, obj *operatorsv1.PackageManifest) *string
	CheckCSV(ctx context.Context, csv string, op *v1alpha1.Operator) (bool, bool)
	DeleteCSV (ctx context.Context, csv string, op *v1alpha1.Operator) error
	DeleteSubscription (ctx context.Context, op *v1alpha1.Operator) error
}

type operatorClient struct {
	kube client.Client
	logger logging.Logger
	client versioned.Interface
}

func NewClient(kube client.Client, logger logging.Logger, p versioned.Interface) Client {
	return operatorClient{kube: kube, logger: logger, client: p}
}

func (o operatorClient) CreateOperator(ctx context.Context, op *v1alpha1.Operator) error {
	sub := operaterv1alpha1.Subscription{
		Spec: &operaterv1alpha1.SubscriptionSpec{
			CatalogSource:          op.Spec.ForProvider.CatalogSource,
			CatalogSourceNamespace: op.Spec.ForProvider.CatalogSourceNamespace,
			Package:                op.Spec.ForProvider.OperatorName,
			Channel:                op.Spec.ForProvider.Channel,
		},
	}
	sub.Namespace = op.Namespace
	sub.Name = op.Name
	return o.kube.Create(ctx, &sub)
}

func (o operatorClient) GetPackageManifest(ctx context.Context, op *v1alpha1.Operator) (*operatorsv1.PackageManifest, error) {
	return o.client.OperatorsV1().PackageManifests(op.Spec.ForProvider.CatalogSourceNamespace).Get(ctx, op.Spec.ForProvider.OperatorName, metav1.GetOptions{})
}

func (o operatorClient) ParsePackageManifest(op *v1alpha1.Operator, obj *operatorsv1.PackageManifest) *string {
	var channel *operatorsv1.PackageChannel = nil
	for _, v := range obj.Status.Channels {
		if v.Name == op.Spec.ForProvider.Channel {
			channel = &v
			break
		}
	}
	o.logger.Debug("ParsePackageManifest", "PM", obj)
	if channel == nil {
		return nil
	}
	return &channel.CurrentCSV
}

func (o operatorClient) CheckCSV(ctx context.Context, csv string, op *v1alpha1.Operator) (bool, bool) {
	cluster := operaterv1alpha1.ClusterServiceVersion{}
	err := o.kube.Get(ctx, client.ObjectKey{
		Namespace: op.Namespace,
		Name:      csv,
	}, &cluster)
	if err != nil {
		return false, false
	}
	return true, cluster.Status.Phase == operaterv1alpha1.CSVPhaseSucceeded
}

func (o operatorClient) DeleteCSV (ctx context.Context, csv string, op *v1alpha1.Operator) error {
	cluster := operaterv1alpha1.ClusterServiceVersion{}
	err := o.kube.Get(ctx, client.ObjectKey{
		Namespace: op.Namespace,
		Name:      csv,
	}, &cluster)
	if err != nil {
		return err
	}
	return o.kube.Delete(ctx, &cluster)
}

func (o operatorClient) DeleteSubscription (ctx context.Context, op *v1alpha1.Operator) error {
	cluster := operaterv1alpha1.Subscription{}
	err := o.kube.Get(ctx, client.ObjectKey{
		Namespace: op.Namespace,
		Name:      op.Name,
	}, &cluster)
	if err != nil {
		return err
	}
	return o.kube.Delete(ctx, &cluster)
}