// Copyright Envoy AI Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestMCPRouteBackendRef_IsMCPBackend(t *testing.T) {
	require.False(t, (&MCPRouteBackendRef{BackendObjectReference: gwapiv1.BackendObjectReference{Name: "svc"}}).IsMCPBackend())
	require.False(t, (&MCPRouteBackendRef{BackendObjectReference: gwapiv1.BackendObjectReference{
		Name: "svc", Kind: ptr.To(gwapiv1.Kind("Service")),
	}}).IsMCPBackend())
	require.False(t, (&MCPRouteBackendRef{BackendObjectReference: gwapiv1.BackendObjectReference{
		Name: "be", Kind: ptr.To(gwapiv1.Kind(EGBackendKind)), Group: ptr.To(gwapiv1.Group(EGBackendGroup)),
	}}).IsMCPBackend())
	require.False(t, (&MCPRouteBackendRef{BackendObjectReference: gwapiv1.BackendObjectReference{
		Name: "be", Kind: ptr.To(gwapiv1.Kind(MCPBackendKind)),
	}}).IsMCPBackend())
	require.True(t, (&MCPRouteBackendRef{BackendObjectReference: gwapiv1.BackendObjectReference{
		Name: "be", Kind: ptr.To(gwapiv1.Kind(MCPBackendKind)), Group: ptr.To(gwapiv1.Group(GroupName)),
	}}).IsMCPBackend())
}

func TestMCPRouteBackendRef_BackendObjectReference(t *testing.T) {
	ref := MCPRouteBackendRef{
		BackendObjectReference: gwapiv1.BackendObjectReference{
			Name: "svc", Namespace: ptr.To(gwapiv1.Namespace("ns")), Port: ptr.To[gwapiv1.PortNumber](80),
		},
	}
	require.Equal(t, gwapiv1.ObjectName("svc"), ref.Name)
	require.Equal(t, ptr.To(gwapiv1.Namespace("ns")), ref.Namespace)
	require.Equal(t, ptr.To[gwapiv1.PortNumber](80), ref.Port)
	require.Nil(t, ref.Group)
	require.Nil(t, ref.Kind)

	ref = MCPRouteBackendRef{
		BackendObjectReference: gwapiv1.BackendObjectReference{
			Name: "eg", Group: ptr.To(gwapiv1.Group(EGBackendGroup)), Kind: ptr.To(gwapiv1.Kind(EGBackendKind)),
		},
	}
	require.Equal(t, ptr.To(gwapiv1.Group(EGBackendGroup)), ref.Group)
	require.Equal(t, ptr.To(gwapiv1.Kind(EGBackendKind)), ref.Kind)
}
