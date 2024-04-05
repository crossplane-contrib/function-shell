package main

import (
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-shell/input/v1beta1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateParameters validates the Parameters object.
func ValidateParameters(p *v1beta1.Parameters, oxr *resource.Composite) *field.Error {
	if p.ShellCommand == "" && p.ShellCommandField == "" {
		return field.Required(field.NewPath("parameters"), "one of ShellCommand or ShellCommandField is required")
	}

	if p.ShellCommand != "" && p.ShellCommandField != "" {
		return field.Required(field.NewPath("parameters"), "exactly one of ShellCommand or ShellCommandField is required")
	}

	return nil
}
