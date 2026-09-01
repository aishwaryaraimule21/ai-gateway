// Copyright Envoy AI Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aigv1b1 "github.com/envoyproxy/ai-gateway/api/v1beta1"
)

// MCPBackendController implements [reconcile.TypedReconciler] for [aigv1b1.MCPBackend].
//
// Exported for testing purposes.
type MCPBackendController struct {
	client       client.Client
	kube         kubernetes.Interface
	logger       logr.Logger
	mcpRouteChan chan event.GenericEvent
}

// NewMCPBackendController creates a new [reconcile.TypedReconciler] for [aigv1b1.MCPBackend].
func NewMCPBackendController(client client.Client, kube kubernetes.Interface, logger logr.Logger, mcpRouteChan chan event.GenericEvent) *MCPBackendController {
	return &MCPBackendController{
		client:       client,
		kube:         kube,
		logger:       logger,
		mcpRouteChan: mcpRouteChan,
	}
}

// Reconcile implements the [reconcile.TypedReconciler] for [aigv1b1.MCPBackend].
func (c *MCPBackendController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	var mcpBackend aigv1b1.MCPBackend
	if err := c.client.Get(ctx, req.NamespacedName, &mcpBackend); err != nil {
		if client.IgnoreNotFound(err) == nil {
			c.logger.Info("Deleting MCPBackend",
				"namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	c.logger.Info("Reconciling MCPBackend", "namespace", req.Namespace, "name", req.Name)
	if err := c.syncMCPBackend(ctx, &mcpBackend); err != nil {
		c.logger.Error(err, "failed to sync MCPBackend")
		c.updateMCPBackendStatus(ctx, &mcpBackend, aigv1b1.ConditionTypeNotAccepted, err.Error())
		return ctrl.Result{}, err
	}
	c.updateMCPBackendStatus(ctx, &mcpBackend, aigv1b1.ConditionTypeAccepted, "MCPBackend reconciled successfully")
	return ctrl.Result{}, nil
}

// syncMCPBackend is the main logic for reconciling the MCPBackend resource.
// This is decoupled from the Reconcile method to centralize the error handling and status updates.
func (c *MCPBackendController) syncMCPBackend(ctx context.Context, mcpBackend *aigv1b1.MCPBackend) error {
	var backendSecurityPolicyList aigv1b1.BackendSecurityPolicyList
	key := fmt.Sprintf("%s.%s", mcpBackend.Name, mcpBackend.Namespace)
	if err := c.client.List(ctx, &backendSecurityPolicyList, client.InNamespace(mcpBackend.Namespace),
		client.MatchingFields{k8sClientIndexMCPBackendToTargetingBackendSecurityPolicy: key}); err != nil {
		return fmt.Errorf("failed to list BackendSecurityPolicyList: %w", err)
	}
	if len(backendSecurityPolicyList.Items) > 1 {
		var names []string
		for i := range backendSecurityPolicyList.Items {
			bsp := &backendSecurityPolicyList.Items[i]
			names = append(names, bsp.Name)
		}
		return fmt.Errorf("multiple BackendSecurityPolicies found for MCPBackend %s: %v",
			mcpBackend.Name, names)
	}

	// Propagate the deletion all the way up to relevant MCPRoutes regardless of being deleted or not.
	_ = handleFinalizer(ctx, c.client, c.logger, mcpBackend, nil)

	var mcpRoutes aigv1b1.MCPRouteList
	err := c.client.List(ctx, &mcpRoutes, client.MatchingFields{k8sClientIndexMCPBackendToReferencingMCPRoute: key})
	if err != nil {
		return fmt.Errorf("failed to list MCPRouteList: %w", err)
	}
	for i := range mcpRoutes.Items {
		mcpRoute := &mcpRoutes.Items[i]
		c.logger.Info("syncing MCPRoute",
			"namespace", mcpRoute.Namespace, "name", mcpRoute.Name,
			"referenced_backend", mcpBackend.Name, "referenced_backend_namespace", mcpBackend.Namespace,
		)
		c.mcpRouteChan <- event.GenericEvent{Object: mcpRoute}
	}
	return nil
}

// updateMCPBackendStatus updates the status of the MCPBackend.
func (c *MCPBackendController) updateMCPBackendStatus(ctx context.Context, backend *aigv1b1.MCPBackend, conditionType string, message string) {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := c.client.Get(ctx, client.ObjectKey{Name: backend.Name, Namespace: backend.Namespace}, backend); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		backend.Status.Conditions = newConditions(conditionType, message)
		return c.client.Status().Update(ctx, backend)
	})
	if err != nil {
		c.logger.Error(err, "failed to update MCPBackend status",
			"namespace", backend.Namespace, "name", backend.Name)
	}
}
