// Copyright Envoy AI Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// MCPBackendKind is the kind of the MCPBackend resource.
	MCPBackendKind = "MCPBackend"
	// EGBackendKind is the kind of the Envoy Gateway Backend resource.
	EGBackendKind = "Backend"
	// EGBackendGroup is the API group of the Envoy Gateway Backend resource.
	EGBackendGroup = "gateway.envoyproxy.io"
)

// MCPBackend is a resource that represents a single upstream MCP server.
// It wraps an Envoy Gateway Backend with MCP-specific configuration including
// the server endpoint path, tool selectors, and header forwarding rules.
//
// An MCPBackend is referenced by MCPRoute.spec.backendRefs[] and can be targeted
// by a BackendSecurityPolicy for upstream authentication configuration.
//
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[-1:].type`
// +kubebuilder:storageversion
type MCPBackend struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines the details of the MCPBackend.
	Spec MCPBackendSpec `json:"spec,omitempty"`
	// Status defines the status details of the MCPBackend.
	Status MCPBackendStatus `json:"status,omitempty"`
}

// MCPBackendList contains a list of MCPBackend.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
type MCPBackendList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCPBackend `json:"items"`
}

// MCPBackendSpec details the MCPBackend configuration.
type MCPBackendSpec struct {
	// BackendRef references the Envoy Gateway Backend (network-level: hostname, port, TLS).
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="has(self.kind) && self.kind == 'Backend' && has(self.group) && self.group == 'gateway.envoyproxy.io'",message="must reference an Envoy Gateway Backend"
	BackendRef gwapiv1.BackendObjectReference `json:"backendRef"`

	// Path is the HTTP endpoint path of the backend MCP server. Defaults to "/mcp".
	//
	// +kubebuilder:default:=/mcp
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Path *string `json:"path,omitempty"`

	// ToolSelector filters the tools exposed by this MCP server.
	// Supports exact matches and RE2-compatible regular expressions for both include and exclude patterns.
	// If not specified, all tools from the MCP server are exposed.
	// +optional
	ToolSelector *MCPToolFilter `json:"toolSelector,omitempty"`

	// ForwardHeaders specifies headers to extract from client requests and forward to this backend.
	//
	// +kubebuilder:validation:MaxItems=32
	// +optional
	ForwardHeaders []MCPHeaderForward `json:"forwardHeaders,omitempty"`
}
