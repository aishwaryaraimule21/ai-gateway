// Copyright Envoy AI Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fake2 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	aigv1b1 "github.com/envoyproxy/ai-gateway/api/v1beta1"
	internaltesting "github.com/envoyproxy/ai-gateway/internal/testing"
)

func TestMCPBackendController_Reconcile(t *testing.T) {
	fakeClient := requireNewFakeClientWithIndexesForMCP(t)
	eventChan := internaltesting.NewControllerEventChan[*aigv1b1.MCPRoute]()
	c := NewMCPBackendController(fakeClient, fake2.NewClientset(), ctrl.Log, eventChan.Ch)

	originals := []*aigv1b1.MCPRoute{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "myroute", Namespace: "default"},
			Spec: aigv1b1.MCPRouteSpec{
				ParentRefs: []gwapiv1.ParentReference{{Name: "gtw"}},
				BackendRefs: []aigv1b1.MCPRouteBackendRef{{
					BackendObjectReference: gwapiv1.BackendObjectReference{
						Name:  "mybackend",
						Group: ptr.To(gwapiv1.Group(aigv1b1.GroupName)),
						Kind:  ptr.To(gwapiv1.Kind(aigv1b1.MCPBackendKind)),
					},
				}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "myroute2", Namespace: "default"},
			Spec: aigv1b1.MCPRouteSpec{
				ParentRefs: []gwapiv1.ParentReference{{Name: "gtw"}},
				BackendRefs: []aigv1b1.MCPRouteBackendRef{{
					BackendObjectReference: gwapiv1.BackendObjectReference{
						Name:  "mybackend",
						Group: ptr.To(gwapiv1.Group(aigv1b1.GroupName)),
						Kind:  ptr.To(gwapiv1.Kind(aigv1b1.MCPBackendKind)),
					},
				}},
			},
		},
	}
	for _, route := range originals {
		require.NoError(t, fakeClient.Create(t.Context(), route))
	}

	err := fakeClient.Create(t.Context(), &aigv1b1.MCPBackend{
		ObjectMeta: metav1.ObjectMeta{Name: "mybackend", Namespace: "default"},
		Spec: aigv1b1.MCPBackendSpec{
			BackendRef: gwapiv1.BackendObjectReference{
				Name:  "eg-backend",
				Kind:  ptr.To(gwapiv1.Kind(aigv1b1.EGBackendKind)),
				Group: ptr.To(gwapiv1.Group(aigv1b1.EGBackendGroup)),
			},
		},
	})
	require.NoError(t, err)
	_, err = c.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "mybackend"}})
	require.NoError(t, err)
	require.Equal(t, originals, eventChan.RequireItemsEventually(t, 2))

	var backend aigv1b1.MCPBackend
	require.NoError(t, fakeClient.Get(t.Context(), types.NamespacedName{Namespace: "default", Name: "mybackend"}, &backend))
	require.Len(t, backend.Status.Conditions, 1)
	require.Equal(t, aigv1b1.ConditionTypeAccepted, backend.Status.Conditions[0].Type)
	require.Equal(t, "MCPBackend reconciled successfully", backend.Status.Conditions[0].Message)
	require.Contains(t, backend.Finalizers, aiGatewayControllerFinalizer, "Finalizer should be set")

	err = fakeClient.Delete(t.Context(), &aigv1b1.MCPBackend{ObjectMeta: metav1.ObjectMeta{Name: "mybackend", Namespace: "default"}})
	require.NoError(t, err)
	_, err = c.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "mybackend"}})
	require.NoError(t, err)
}

func TestMCPBackendController_Reconcile_error_with_multiple_bsps(t *testing.T) {
	fakeClient := requireNewFakeClientWithIndexesForMCP(t)
	eventChan := internaltesting.NewControllerEventChan[*aigv1b1.MCPRoute]()
	c := NewMCPBackendController(fakeClient, fake2.NewClientset(), ctrl.Log, eventChan.Ch)

	const backendName, namespace = "mybackend", "default"
	for i := range 3 {
		bsp := &aigv1b1.BackendSecurityPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("bsp-%d", i), Namespace: namespace},
			Spec: aigv1b1.BackendSecurityPolicySpec{
				Type: aigv1b1.BackendSecurityPolicyTypeMCPAPIKey,
				MCPAPIKey: &aigv1b1.MCPBackendAPIKey{
					Inline: ptr.To("key"),
				},
				TargetRefs: []gwapiv1a2.LocalPolicyTargetReference{{
					Group: aigv1b1.GroupName,
					Kind:  aigv1b1.MCPBackendKind,
					Name:  gwapiv1.ObjectName(backendName),
				}},
			},
		}
		require.NoError(t, fakeClient.Create(t.Context(), bsp))
	}

	err := fakeClient.Create(t.Context(), &aigv1b1.MCPBackend{ObjectMeta: metav1.ObjectMeta{Name: backendName, Namespace: namespace}})
	require.NoError(t, err)
	_, err = c.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: backendName}})
	require.ErrorContains(t, err, `multiple BackendSecurityPolicies found for MCPBackend mybackend: [bsp-0 bsp-1 bsp-2]`)
}
