package main

import (
	"github.com/crossplane-contrib/function-shell/input/v1alpha1"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

func getCredentialData(req *fnv1.RunFunctionRequest, credName string) map[string][]byte {
	var data map[string][]byte
	switch req.GetCredentials()[credName].GetSource().(type) {
	case *fnv1.Credentials_CredentialData:
		data = req.GetCredentials()[credName].GetCredentialData().GetData()
	default:
		return nil
	}

	return data
}

func addShellEnvVarsFromCredentialRefs(req *fnv1.RunFunctionRequest, credentialRefs v1alpha1.ShellCredentialRef, shellEnvVars map[string]string) (map[string]string, error) {
	var credentialData = getCredentialData(req, credentialRefs.Name)
	for _, key := range credentialRefs.Keys {
		shellEnvVars[key] = string(credentialData[key])
	}
	return shellEnvVars, nil
}
