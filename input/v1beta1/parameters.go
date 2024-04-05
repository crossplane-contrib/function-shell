// Package v1beta1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=template.fn.crossplane.io
// +versionName=v1beta1
package v1beta1

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

	// shellEnvVarSecretRef
	ShellEnvVarsSecretRef ShellEnvVarsSecretRef `json:"shellEnvVarsSecretRef,omitempty"`

	// shellEnvVars
	ShellEnvVars []ShellEnvVar `json:"shellEnvVars,omitempty"`

	// shellCmd
	ShellCommand string `json:"shellCommand,omitempty"`

	// shellCmdField
	ShellCommandField string `json:"shellCommandField,omitempty"`

	// stdoutField
	// +optional
	StdoutField string `json:"stdoutField,omitempty"`

	// stderrField
	// +optional
	StderrField string `json:"stderrField,omitempty"`
}

type ShellEnvVar struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type ShellEnvVarsSecretRef struct {
	Key       string `json:"key,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}
