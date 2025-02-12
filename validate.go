package main

import (
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/giantswarm/function-shell-idp/input/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateParameters validates the Parameters object.
func ValidateParameters(p *v1alpha1.Parameters, oxr *resource.Composite) *field.Error {
	if p.ShellCommand == "" && p.ShellCommandField == "" {
		return field.Required(field.NewPath("parameters"), "one of ShellCommand or ShellCommandField is required")
	}

	if p.ShellCommand != "" && p.ShellCommandField != "" {
		return field.Required(field.NewPath("parameters"), "exactly one of ShellCommand or ShellCommandField is required")
	}

	return nil
}
