package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/crossplane-contrib/function-shell/input/v1alpha1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/fieldpath"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
)

func addShellEnvVarsFromRef(envVarsRef v1alpha1.ShellEnvVarsRef, shellEnvVars map[string]string) (map[string]string, error) {
	var envVarsData map[string]string

	envVars := os.Getenv(envVarsRef.Name)
	if err := json.Unmarshal([]byte(envVars), &envVarsData); err != nil {
		return shellEnvVars, err
	}
	for _, key := range envVarsRef.Keys {
		shellEnvVars[key] = envVarsData[key]
	}
	return shellEnvVars, nil
}

func fromFieldRef(req *fnv1.RunFunctionRequest, fieldRef v1alpha1.FieldRef) (string, error) {
	if fieldRef.Path == "" {
		return "", errors.New("path must be set")
	}
	// Check for context key presence and capture context key and path
	contextRegex := regexp.MustCompile(`^context\[(.+?)].(.+)$`)
	if match := contextRegex.FindStringSubmatch(fieldRef.Path); match != nil {
		if v, ok := request.GetContextKey(req, match[1]); ok {
			context := &unstructured.Unstructured{}
			if err := resource.AsObject(v.GetStructValue(), context); err != nil {
				return "", errors.Wrapf(err, "cannot convert context to %s", v)
			}
			value, err := fieldpath.Pave(context.Object).GetValue(match[2])
			if err != nil {
				switch fieldRef.Policy {
				case v1alpha1.FieldRefPolicyOptional:
					return fieldRef.DefaultValue, nil
				case v1alpha1.FieldRefPolicyRequired:
					fallthrough
				default:
					return "", errors.Wrap(err, "cannot get context value")
				}
			}
			return fmt.Sprintf("%v", value), nil
		}
	} else {
		oxr, err := request.GetObservedCompositeResource(req)
		if err != nil {
			return "", errors.Wrapf(err, "cannot get observed composite resource from %T", req)
		}
		value, err := oxr.Resource.GetValue(fieldRef.Path)
		if err != nil {
			switch fieldRef.Policy {
			case v1alpha1.FieldRefPolicyOptional:
				return fieldRef.DefaultValue, nil
			case v1alpha1.FieldRefPolicyRequired:
				fallthrough
			default:
				return "", errors.Wrap(err, "cannot get observed composite value")
			}
		}
		return fmt.Sprintf("%v", value), nil
	}
	return fieldRef.DefaultValue, nil
}

// a valueRef behaves like a fieldRef with a Required Policy.
func fromValueRef(req *fnv1.RunFunctionRequest, path string) (string, error) {
	return fromFieldRef(
		req, v1alpha1.FieldRef{
			Path:   path,
			Policy: v1alpha1.FieldRefPolicyRequired,
		})
}
