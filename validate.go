package main

import (
	"github.com/crossplane-contrib/function-shell/input/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/crossplane/function-sdk-go/resource"
)

// ValidateParameters validates the Parameters object.
func ValidateParameters(p *v1alpha1.Parameters, _ *resource.Composite) *field.Error {
	if p.ShellCommand == "" && p.ShellCommandField == "" {
		return field.Required(field.NewPath("parameters"), "one of ShellCommand or ShellCommandField is required")
	}

	if p.ShellCommand != "" && p.ShellCommandField != "" {
		return field.Required(field.NewPath("parameters"), "exactly one of ShellCommand or ShellCommandField is required")
	}

	return nil
}
