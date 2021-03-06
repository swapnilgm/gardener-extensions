// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gardener/gardener-extensions/pkg/util"

	"github.com/gardener/gardener-resource-manager/pkg/manager"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RenderChartAndCreateManagedResource renders a chart and creates a ManagedResource for the gardener-resource-manager
// out of the results.
func RenderChartAndCreateManagedResource(
	ctx context.Context,
	namespace string,
	name string,
	client client.Client,
	chartRenderer chartrenderer.Interface,
	chart util.Chart,
	values map[string]interface{},
	imageVector imagevector.ImageVector,
	chartNamespace string,
	version string,
	withNoCleanupLabel bool,
) error {
	chartName, data, err := chart.Render(chartRenderer, chartNamespace, imageVector, version, version, values)
	if err != nil {
		return errors.Wrapf(err, "could not render chart")
	}

	// Create or update managed resource referencing the previously created secret
	var injectedLabels map[string]string
	if withNoCleanupLabel {
		injectedLabels = map[string]string{ShootNoCleanupLabel: "true"}
	}

	return CreateManagedResource(ctx, client, namespace, name, "", chartName, data, false, injectedLabels)
}

func CreateManagedResourceFromUnstructured(ctx context.Context, client client.Client, namespace, name, class string, objs []*unstructured.Unstructured, keepObjects bool, injectedLabels map[string]string) error {
	var data []byte
	for _, obj := range objs {
		bytes, err := obj.MarshalJSON()
		if err != nil {
			return errors.Wrapf(err, "marshal failed for '%s/%s' for secret '%s/%s'", obj.GetNamespace(), obj.GetName(), namespace, name)
		}
		data = append(data, []byte("\n---\n")...)
		data = append(data, bytes...)
	}
	return CreateManagedResource(ctx, client, namespace, name, class, name, data, keepObjects, injectedLabels)
}

func CreateManagedResourceFromFileChart(ctx context.Context, client client.Client, namespace, name, class string, renderer chartrenderer.Interface, chartPath, chartName string, chartValues map[string]interface{}, injectedLabels map[string]string) error {
	chart, err := renderer.Render(chartPath, chartName, namespace, chartValues)
	if err != nil {
		return err
	}

	return CreateManagedResource(ctx, client, namespace, name, class, chartName, chart.Manifest(), false, injectedLabels)
}

func CreateManagedResource(ctx context.Context, client client.Client, namespace, name, class, key string, data []byte, keepObjects bool, injectedLabels map[string]string) error {
	if key == "" {
		key = name
	}

	// Create or update secret containing the rendered rbac manifests
	if err := manager.NewSecret(client).
		WithNamespacedName(namespace, name).
		WithKeyValues(map[string][]byte{key: data}).
		Reconcile(ctx); err != nil {
		return errors.Wrapf(err, "could not create or update secret '%s/%s' of managed resources", namespace, name)
	}

	if err := manager.NewManagedResource(client).
		WithNamespacedName(namespace, name).
		WithClass(class).
		WithInjectedLabels(injectedLabels).
		KeepObjects(keepObjects).
		WithSecretRef(name).
		Reconcile(ctx); err != nil {
		return errors.Wrapf(err, "could not create or update managed resource '%s/%s'", namespace, name)
	}

	return nil
}

// DeleteManagedResource deletes a managed resource and a secret with the given <name>.
func DeleteManagedResource(ctx context.Context, client client.Client, namespace string, name string) error {
	if err := manager.
		NewManagedResource(client).
		WithNamespacedName(namespace, name).
		Delete(ctx); err != nil {
		return errors.Wrapf(err, "could not delete managed resource '%s/%s'", namespace, name)
	}

	if err := manager.
		NewSecret(client).
		WithNamespacedName(namespace, name).
		Delete(ctx); err != nil {
		return errors.Wrapf(err, "could not delete secret '%s/%s' of managed resource", namespace, name)
	}

	return nil
}
