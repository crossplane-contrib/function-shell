package main

import (
	"testing"

	"github.com/crossplane-contrib/function-shell/input/v1alpha1"
	"github.com/google/go-cmp/cmp"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
)

func TestGetCredentialData(t *testing.T) {
	type args struct {
		req *fnv1.RunFunctionRequest
	}

	type want struct {
		data map[string]string
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"RetrieveFunctionCredential": {
			reason: "Should successfully retrieve the function credential",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCredentialRefs": [{"name": "foo-creds", "keys": ["password"]}],
						"shellEnvVars": [{"key": "TEST_ENV_VAR", "value": "foo"}],
						"shellCommand": "echo ${TEST_ENV_VAR}",
						"stdoutField": "spec.atFunction.shell.stdout"
					}`),
					Credentials: map[string]*fnv1.Credentials{
						"foo-creds": {
							Source: &fnv1.Credentials_CredentialData{
								CredentialData: &fnv1.CredentialData{
									Data: map[string][]byte{
										"password": []byte("secret"),
									},
								},
							},
						},
					},
				},
			},
			want: want{
				data: map[string]string{
					"password": "secret",
				},
			},
		},
		"FunctionCredentialNotFound": {
			reason: "Should return nil if the function credential is not found",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCredentialRefs": [{"name": "foo-creds", "keys": ["password"]}],
						"shellEnvVars": [{"key": "TEST_ENV_VAR", "value": "foo"}],
						"shellCommand": "echo ${TEST_ENV_VAR}",
						"stdoutField": "spec.atFunction.shell.stdout"
					}`),
					Credentials: map[string]*fnv1.Credentials{},
				},
			},
			want: want{
				data: map[string]string{
					"password": "",
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, _ := addShellEnvVarsFromCredentialRefs(tc.args.req, v1alpha1.ShellCredentialRef{Name: "foo-creds", Keys: []string{"password"}}, map[string]string{})
			if diff := cmp.Diff(tc.want.data, got); diff != "" {
				t.Errorf("%s\naddShellEnvVarsFromCredentialRefs(...): -want data, +got data:\n%s", tc.reason, diff)
			}
		})
	}
}
