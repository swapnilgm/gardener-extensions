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

package backupentry

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Actuator acts upon BackupEntry resources.
type Actuator interface {
	// Reconcile reconciles the BackupEntry.
	Reconcile(context.Context, *extensionsv1alpha1.BackupEntry) error
	// Delete deletes the BackupEntry.
	Delete(context.Context, *extensionsv1alpha1.BackupEntry) (bool, error)
}

// ProviderActuator acts upon BackupEntry resources.
type ProviderActuator interface {
	inject.Client
	// Reconcile reconciles the BackupEntry.
	Reconcile(context.Context, *extensionsv1alpha1.BackupEntry) error
	// Delete deletes the BackupEntry.
	Delete(context.Context, *extensionsv1alpha1.BackupEntry) error
}

type actuator struct {
	ProviderActuator
	logger                  logr.Logger
	deletionGracePeriodDays *int64
}

// NewActuator creates a new Actuator that updates the status of the handled Infrastructure resources.
func NewActuator(providerActuator ProviderActuator, deletionGracePeriodDays *int64, logger logr.Logger) Actuator {
	return &actuator{
		logger:                  logger.WithName("backupentry-actuator"),
		ProviderActuator:        providerActuator,
		deletionGracePeriodDays: deletionGracePeriodDays,
	}
}

func (a *actuator) InjectClient(client client.Client) error {
	return a.ProviderActuator.InjectClient(client)
}

// Reconcile reconciles the update of a BackupEntry
func (a *actuator) Reconcile(ctx context.Context, be *extensionsv1alpha1.BackupEntry) error {
	return a.ProviderActuator.Reconcile(ctx, be)
}

// Delete delete the BackupEntry
func (a *actuator) Delete(ctx context.Context, be *extensionsv1alpha1.BackupEntry) (bool, error) {
	gracePeriod := computeGracePeriod(a.deletionGracePeriodDays)
	if time.Since(be.DeletionTimestamp.Time) > gracePeriod {
		if err := a.ProviderActuator.Delete(ctx, be); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func computeGracePeriod(deletionGracePeriodDays *int64) time.Duration {
	if deletionGracePeriodDays == nil {
		return time.Duration(0)
	}
	return time.Hour * 24 * time.Duration(*deletionGracePeriodDays)
}
