// Package v1alpha1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=shell.fn.crossplane.giantswarm.io
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
