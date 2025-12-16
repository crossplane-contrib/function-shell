// Package v1alpha1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=template.fn.crossplane.io
// +versionName=v1alpha1
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This isn't a custom resource, in the sense that we never install its CRD.
// It is a KRM-like object, so we generate a CRD to describe its schema.

// Parameters can be used to provide input to this Function.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane
type Parameters struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// shellEnvVarsRef
	// +optional
	ShellEnvVarsRef ShellEnvVarsRef `json:"shellEnvVarsRef"`

	// shellEnvVars
	// +optional
	ShellEnvVars []ShellEnvVar `json:"shellEnvVars"`

	// shellCmd
	// +optional
	ShellCommand string `json:"shellCommand"`

	// shellCmdField
	// +optional
	ShellCommandField string `json:"shellCommandField,omitempty"`

	// stdoutField
	// +optional
	StdoutField string `json:"stdoutField,omitempty"`

	// stderrField
	// +optional
	StderrField string `json:"stderrField,omitempty"`

	// TTL for response cache. Function Response caching is an
	// alpha feature in Crossplane can be deprecated or changed
	// in the future.
	// +optional
	// +kubebuilder:default:="1m"
	CacheTTL string `json:"cacheTTL,omitempty"`
}

// ShellEnvVarType is a type of ShellEnvVar.
type ShellEnvVarType string

const (
	// ShellEnvVarTypeFieldRef is a reference to a field in the Composition.
	ShellEnvVarTypeFieldRef ShellEnvVarType = "FieldRef"
	// ShellEnvVarTypeValue populates the value from a string.
	ShellEnvVarTypeValue ShellEnvVarType = "Value"
	// ShellEnvVarTypeValueRef is a reference to a field in the Composition.
	ShellEnvVarTypeValueRef ShellEnvVarType = "ValueRef"
)

// ShellEnvVar is a Shell Environment Variable of the form key=value.
type ShellEnvVar struct {
	// Key is the Environment Variable key like API_KEY
	Key string `json:"key,omitempty"`
	// Value is a fixed value, like http://api.example.com
	Value string `json:"value,omitempty"`
	// ValueRef retrieves a Environment Variable value from a composite field.
	// Can result in error if field is not set: use FieldRef which can handle missing fields.
	ValueRef string `json:"valueRef,omitempty"`
	// FieldRef is a reference to a field in the Composition.
	FieldRef *FieldRef `json:"fieldRef,omitempty"`
	// Type is the type of ShellEnVar: Value, ValueRef, FieldRef.
	Type ShellEnvVarType `json:"type,omitempty"`
}

// GetType determines the ShellEnvVar type.
func (sev *ShellEnvVar) GetType() ShellEnvVarType {
	if sev.Type == "" {
		if sev.Value != "" {
			return ShellEnvVarTypeValue
		}
		if sev.ValueRef != "" {
			return ShellEnvVarTypeValueRef
		}
		if sev.FieldRef != nil {
			return ShellEnvVarTypeFieldRef
		}
	}
	return sev.Type
}

// ShellEnvVarsRef refers to an environment variable or secret leaded into
// the function pod.
type ShellEnvVarsRef struct {
	// The Key whose value is the secret
	Keys []string `json:"keys,omitempty"`
	// Name of the environment variable
	Name string `json:"name,omitempty"`
}

// FieldRefPolicy is a field path Policy.
type FieldRefPolicy string

// FieldRefPolicyOptional if the field is not available use the value of FieldRefDefault.
const FieldRefPolicyOptional = "Optional"

// FieldRefPolicyRequired will error if the field is not available.
const FieldRefPolicyRequired = "Required"

// FieldRefDefault optional value result returned for an Optional FieldRef. Defaults to an empty string.
const FieldRefDefault = ""

// FieldRef refers to a composite field like spec.region.
type FieldRef struct {
	// Path is the field path of the field being referenced, i.e. spec.myfield, status.output
	Path string `json:"path"`
	// Policy when the field is not available. If set to "Required" will return
	// an error if a field is missing. If set to "Optional" will return DefaultValue.
	// +optional
	// +kubebuilder:default:=Required
	// +kubebuilder:validation:Enum=Optional;Required
	Policy FieldRefPolicy `json:"policy,omitempty"`
	// DefaultValue when Policy is Optional and field is not available defaults to ""
	// +optional
	// +kbuebuilder:default:=""
	DefaultValue string `json:"defaultValue,omitempty"`
}
