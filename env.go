package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/giantswarm/function-shell-idp/input/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func fromValueRef(req *fnv1beta1.RunFunctionRequest, path string) (string, error) {
	// Check for context key presence and capture context key and path
	contextRegex := regexp.MustCompile(`^context\[(.+?)].(.+)$`)
	if match := contextRegex.FindStringSubmatch(path); match != nil {
		if v, ok := request.GetContextKey(req, match[1]); ok {
			context := &unstructured.Unstructured{}
			if err := resource.AsObject(v.GetStructValue(), context); err != nil {
				return "", errors.Wrapf(err, "cannot convert context to %s", v)
			}
			value, err := fieldpath.Pave(context.Object).GetValue(match[2])
			if err != nil {
				return "", errors.Wrapf(err, "cannot get context value at %s", match[2])
			}
			return fmt.Sprintf("%v", value), nil
		}
	} else {
		oxr, err := request.GetObservedCompositeResource(req)
		if err != nil {
			return "", errors.Wrapf(err, "cannot get observed composite resource from %T", req)
		}
		value, err := oxr.Resource.GetValue(path)
		if err != nil {
			return "", errors.Wrapf(err, "cannot get observed composite value at %s", path)
		}
		return fmt.Sprintf("%v", value), nil

	}
	return "", nil
}
