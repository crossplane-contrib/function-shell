package main

import (
	"encoding/json"
	"os"

	"github.com/crossplane-contrib/function-shell/input/v1alpha1"
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
