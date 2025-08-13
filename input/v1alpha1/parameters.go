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

// Input can be used to provide input to this Function.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane
type Parameters struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// FieldRef is a reference to a field in the Composition
	FieldRef FieldRef `json:"fieldRef,omitempty"`

	// shellEnvVarsRef
	// +optional
	ShellEnvVarsRef ShellEnvVarsRef `json:"shellEnvVarsRef,omitempty"`

	// shellEnvVars
	// +optional
	ShellEnvVars []ShellEnvVar `json:"shellEnvVars,omitempty"`

	// shellCmd
	// +optional
	ShellCommand string `json:"shellCommand,omitempty"`

	// shellCmdField
	// +optional
	ShellCommandField string `json:"shellCommandField,omitempty"`

	// stdoutField
	// +optional
	StdoutField string `json:"stdoutField,omitempty"`

	// stderrField
	// +optional
	StderrField string `json:"stderrField,omitempty"`
}

type ShellEnvVar struct {
	Key      string `json:"key,omitempty"`
	Value    string `json:"value,omitempty"`
	ValueRef string `json:"valueRef,omitempty"`
}

type ShellEnvVarsRef struct {
	// The Key whose value is the secret
	Keys []string `json:"keys,omitempty"`
	// Name of the enviroment variable
	Name string `json:"name,omitempty"`
}

type FieldRefPolicy string

// FieldRefPolicyOptional if the field is not available use the value of FieldRefDefault
const FieldRefPolicyOptional = "Optional"

// FieldRefPolicyRequired will error if the field is not available
const FieldRefPolicyRequired = "Required"

// FieldRefDefault optional value result returned for an Optional FieldRef. Defaults to an empty string

const FieldRefDefault = ""

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
