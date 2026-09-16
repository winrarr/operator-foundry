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
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	patternsv1alpha1 "github.com/winrarr/operator-foundry/api/patterns/v1alpha1"
)

// PatternConnectionReconciler demonstrates a connection health check and
// Secret watch. New operators may replace the authentication model while
// retaining the dependency and status patterns.
type PatternConnectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=patterns.operator-foundry.example,resources=patternconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=patterns.operator-foundry.example,resources=patternconnections/status,verbs=get;patch;update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *PatternConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var connection patternsv1alpha1.PatternConnection
	if err := r.Get(ctx, req.NamespacedName, &connection); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	before := connection.DeepCopy()
	connection.Status.ObservedGeneration = connection.Generation
	apiClient, err := externalClientForConnection(ctx, r.Client, &connection)
	if err != nil {
		if isDependencyError(err) {
			markReconciling(&connection.Status.Conditions, connection.Generation, "DependencyNotReady", err.Error())
			return ctrl.Result{RequeueAfter: DependencyRetry}, persistStatus(ctx, r.Client, &connection, before)
		}
		markStalled(&connection.Status.Conditions, connection.Generation, "InvalidConfiguration", err)
		return ctrl.Result{}, persistStatus(ctx, r.Client, &connection, before)
	}

	if err := apiClient.Health(ctx); err != nil {
		markError(&connection.Status.Conditions, connection.Generation, "ConnectionUnavailable", err)
		if statusErr := persistStatus(ctx, r.Client, &connection, before); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{RequeueAfter: ExternalRetry}, nil
	}

	checkedAt := metav1.Now()
	connection.Status.LastCheckedAt = &checkedAt
	markReady(&connection.Status.Conditions, connection.Generation, "external API is reachable")
	return ctrl.Result{}, persistStatus(ctx, r.Client, &connection, before)
}

func (r *PatternConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&patternsv1alpha1.PatternConnection{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.mapSecretToConnections)).
		Complete(r)
}

func (r *PatternConnectionReconciler) mapSecretToConnections(ctx context.Context, object client.Object) []reconcile.Request {
	var connections patternsv1alpha1.PatternConnectionList
	if err := r.List(ctx, &connections, client.InNamespace(object.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range connections.Items {
		connection := &connections.Items[i]
		if strings.TrimSpace(connection.Spec.AuthSecretRef.Name) == object.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: connection.Name, Namespace: connection.Namespace}})
		}
	}
	return requests
}
