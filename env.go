package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/crossplane-contrib/function-shell/input/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	ExtraResourceFunctionContextKeyEnvironment = "apiextensions.crossplane.io/extra-resources"
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

func fromExtraResourceField(req *fnv1beta1.RunFunctionRequest, path string) (value string, err error) {
	var extraResources *unstructured.Unstructured
	if v, ok := request.GetContextKey(req, ExtraResourceFunctionContextKeyEnvironment); ok {
		extraResources = &unstructured.Unstructured{}
		if err = resource.AsObject(v.GetStructValue(), extraResources); err != nil {
			return
		}
		valueRaw, errValue := fieldpath.Pave(extraResources.Object).GetValue(path)
		if errValue != nil {
			return
		}
		return fmt.Sprintf("%v", valueRaw), nil
	}
	return
}
