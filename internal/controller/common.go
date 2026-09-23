/*
Copyright 2026 winrarr.

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

package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	patternsv1alpha1 "github.com/winrarr/operator-foundry/api/patterns/v1alpha1"
	"github.com/winrarr/operator-foundry/internal/exampleclient"
)

const (
	ConditionReady       = "Ready"
	ConditionReconciling = "Reconciling"
	ConditionStalled     = "Stalled"
	FinalizerName        = "patterns.operator-foundry.example/finalizer"
	DependencyRetry      = 15 * time.Second
	ExternalRetry        = 30 * time.Second
)

type dependencyError struct {
	message string
	cause   error
}

func (e *dependencyError) Error() string { return e.message }
func (e *dependencyError) Unwrap() error { return e.cause }

func newDependencyError(format string, args ...any) error {
	return &dependencyError{message: fmt.Sprintf(format, args...)}
}

func newMissingDependencyError(cause error, format string, args ...any) error {
	return &dependencyError{message: fmt.Sprintf(format, args...), cause: cause}
}

func isDependencyError(err error) bool {
	var dependencyErr *dependencyError
	return errors.As(err, &dependencyErr)
}

func setCondition(conditions *[]metav1.Condition, generation int64, conditionType string, status metav1.ConditionStatus, reason, message string) {
	for i := range *conditions {
		condition := &(*conditions)[i]
		if condition.Type != conditionType {
			continue
		}
		if condition.Status == status && condition.Reason == reason && condition.Message == message && condition.ObservedGeneration == generation {
			return
		}
		transitionTime := metav1.Now()
		if condition.Status == status && !condition.LastTransitionTime.IsZero() {
			transitionTime = condition.LastTransitionTime
		}
		*condition = metav1.Condition{
			Type:               conditionType,
			Status:             status,
			ObservedGeneration: generation,
			LastTransitionTime: transitionTime,
			Reason:             reason,
			Message:            message,
		}
		return
	}
	*conditions = append(*conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func markReady(conditions *[]metav1.Condition, generation int64, message string) {
	setCondition(conditions, generation, ConditionReady, metav1.ConditionTrue, "Reconciled", message)
	setCondition(conditions, generation, ConditionReconciling, metav1.ConditionFalse, "ReconciliationSucceeded", "reconciliation completed")
	setCondition(conditions, generation, ConditionStalled, metav1.ConditionFalse, "NotStalled", "reconciliation can continue")
}

func markReconciling(conditions *[]metav1.Condition, generation int64, reason, message string) {
	setCondition(conditions, generation, ConditionReady, metav1.ConditionFalse, reason, message)
	setCondition(conditions, generation, ConditionReconciling, metav1.ConditionTrue, "Progressing", message)
	setCondition(conditions, generation, ConditionStalled, metav1.ConditionFalse, "NotStalled", "reconciliation can continue")
}

func markStalled(conditions *[]metav1.Condition, generation int64, reason string, err error) {
	message := "reconciliation is stalled"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = err.Error()
	}
	setCondition(conditions, generation, ConditionReady, metav1.ConditionFalse, reason, message)
	setCondition(conditions, generation, ConditionReconciling, metav1.ConditionFalse, "Stalled", "reconciliation is blocked until the resource is corrected")
	setCondition(conditions, generation, ConditionStalled, metav1.ConditionTrue, reason, message)
}

func markError(conditions *[]metav1.Condition, generation int64, reason string, err error) {
	markReconciling(conditions, generation, reason, fmt.Sprintf("reconciliation failed: %s", err))
}

func persistStatus(ctx context.Context, kubeClient client.Client, object client.Object, before client.Object) error {
	if reflect.DeepEqual(before, object) {
		return nil
	}
	return kubeClient.Status().Patch(ctx, object, client.MergeFrom(before))
}

func ensureFinalizer(ctx context.Context, kubeClient client.Client, object client.Object) error {
	if controllerutil.ContainsFinalizer(object, FinalizerName) {
		return nil
	}
	controllerutil.AddFinalizer(object, FinalizerName)
	return kubeClient.Update(ctx, object)
}

func removeFinalizer(ctx context.Context, kubeClient client.Client, object client.Object) error {
	if !controllerutil.ContainsFinalizer(object, FinalizerName) {
		return nil
	}
	controllerutil.RemoveFinalizer(object, FinalizerName)
	return kubeClient.Update(ctx, object)
}

func removeFinalizerAfterDependencyLoss(ctx context.Context, kubeClient client.Client, object client.Object, dependency string, err error) (ctrl.Result, error) {
	var dependencyErr *dependencyError
	dependencyMissing := apierrors.IsNotFound(err) ||
		(errors.As(err, &dependencyErr) && apierrors.IsNotFound(dependencyErr.cause))
	if !dependencyMissing {
		return ctrl.Result{}, err
	}
	log.FromContext(ctx).Error(err, "releasing deletion finalizer because cleanup dependency is unavailable; external state may be orphaned", "dependency", dependency, "resource", client.ObjectKeyFromObject(object))
	return ctrl.Result{}, removeFinalizer(ctx, kubeClient, object)
}

func externalName(name, configured string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	return name
}

func requeueFor(interval *metav1.Duration, fallback time.Duration) time.Duration {
	if interval == nil {
		return fallback
	}
	if interval.Duration <= 0 {
		return 0
	}
	return interval.Duration
}

func getConnection(ctx context.Context, kubeClient client.Client, namespace, name string) (*patternsv1alpha1.PatternConnection, error) {
	if strings.TrimSpace(name) == "" {
		return nil, newDependencyError("connectionRef.name is required")
	}
	var connection patternsv1alpha1.PatternConnection
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &connection); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, newMissingDependencyError(err, "PatternConnection %s/%s was not found", namespace, name)
		}
		return nil, fmt.Errorf("get PatternConnection %s/%s: %w", namespace, name, err)
	}
	return &connection, nil
}

func externalClientForConnection(ctx context.Context, kubeClient client.Client, connection *patternsv1alpha1.PatternConnection) (*exampleclient.Client, error) {
	key := connection.Spec.AuthSecretRef.Key
	if key == "" {
		key = "token"
	}
	var secret corev1.Secret
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: connection.Spec.AuthSecretRef.Name, Namespace: connection.Namespace}, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, newMissingDependencyError(err, "authentication Secret %s/%s was not found", connection.Namespace, connection.Spec.AuthSecretRef.Name)
		}
		return nil, fmt.Errorf("get authentication Secret %s/%s: %w", connection.Namespace, connection.Spec.AuthSecretRef.Name, err)
	}
	token := strings.TrimSpace(string(secret.Data[key]))
	if token == "" {
		return nil, newDependencyError("authentication Secret %s/%s does not contain a non-empty %q key", connection.Namespace, connection.Spec.AuthSecretRef.Name, key)
	}
	timeout := 30 * time.Second
	if connection.Spec.RequestTimeout != nil {
		if connection.Spec.RequestTimeout.Duration <= 0 {
			return nil, fmt.Errorf("requestTimeout must be greater than zero")
		}
		timeout = connection.Spec.RequestTimeout.Duration
	}
	return exampleclient.New(connection.Spec.Endpoint, token, timeout)
}
