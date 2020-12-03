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

// Package apis contains Kubernetes APIs for In-Cluster Provider.
package apis

import (
	fwv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"k8s.io/apimachinery/pkg/runtime"

	databasev1alpha1 "github.com/crossplane-contrib/provider-in-cluster/apis/database/v1alpha1"
	operatorv1alpha1 "github.com/crossplane-contrib/provider-in-cluster/apis/operator/v1alpha1"
	"github.com/crossplane-contrib/provider-in-cluster/apis/v1beta1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		v1beta1.SchemeBuilder.AddToScheme,
		databasev1alpha1.SchemeBuilder.AddToScheme,
		operatorv1alpha1.SchemeBuilder.AddToScheme,
		operatorsv1.SchemeBuilder.AddToScheme,
		fwv1alpha1.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
