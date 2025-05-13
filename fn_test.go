package main

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
)

func TestRunFunction(t *testing.T) {
	type args struct {
		ctx context.Context
		req *fnv1beta1.RunFunctionRequest
	}
	type want struct {
		rsp *fnv1beta1.RunFunctionResponse
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ResponseIsParametersRequired": {
			reason: "The Function should return a fatal result if no input was specified",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Meta: &fnv1beta1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters"
					}`),
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1beta1.Result{
						{
							Severity: fnv1beta1.Severity_SEVERITY_FATAL,
							Message:  "invalid Function input: parameters: Required value: one of ShellCommand or ShellCommandField is required",
						},
					},
				},
			},
		},
		"ResponseIsEmptyShellCommand": {
			reason: "The Function should return a response when after a script is run",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Meta: &fnv1beta1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": ""
					}`),
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1beta1.Result{
						{
							Severity: fnv1beta1.Severity_SEVERITY_FATAL,
							Message:  "invalid Function input: parameters: Required value: one of ShellCommand or ShellCommandField is required",
						},
					},
				},
			},
		},
		"ResponseIsEcho": {
			reason: "The Function should write stdout to the specified field",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Meta: &fnv1beta1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": "echo foo",
						"stdoutField": "spec.atFunction.shell.stdout"
					}`),
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"spec": {
									"atFunction": {
										"shell": {
											"stdout": "foo"
										}
									}
								},
								"status": {
									"atFunction": {
										"shell": {
											"stderr": ""
										}
									}
								}
							}`),
						},
					},
				},
			},
		},
		"ResponseIsErrorIfInvalidShellCommand": {
			reason: "The function should write to the specified stderr when the shell command is invalid",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Meta: &fnv1beta1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": "set -euo pìpefail",
						"stdoutField": "status.atFunction.shell.stdout",
						"stderrField": "status.atFunction.shell.stderr"
                    }`),
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"status": {
									"atFunction": {
										"shell": {
                                            "stdout": "",
											"stderr": "/bin/sh: 1: set: Illegal option -o pìpefail"
										}
									}
								}
							}`),
						},
					},
					Results: []*fnv1beta1.Result{
						{
							Severity: fnv1beta1.Severity_SEVERITY_FATAL,
							Message:  "shellCmd \"set -euo pìpefail\" for \"\" failed with : exit status 2",
						},
					},
				},
			},
		},
		"ResponseIsErrorWhenShellCommandNotFound": {
			reason: "The Function should write to the specified stderr field when the shellCommand is not found",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Meta: &fnv1beta1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": "unkown-shell-command",
						"stdoutField": "status.atFunction.shell.stdout",
						"stderrField": "status.atFunction.shell.stderr"
					}`),
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"status": {
									"atFunction": {
										"shell": {
                                            "stdout": "",
											"stderr": "/bin/sh: 1: unkown-shell-command: not found"
										}
									}
								}
							}`),
						},
					},
					Results: []*fnv1beta1.Result{
						{
							Severity: fnv1beta1.Severity_SEVERITY_FATAL,
							Message:  "shellCmd \"unkown-shell-command\" for \"\" failed with : exit status 127",
						},
					},
				},
			},
		},
		"ResponseIsEchoEnvVar": {
			reason: "The Function should accept and use environment variables",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Meta: &fnv1beta1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellEnvVars": [{"key": "TEST_ENV_VAR", "value": "foo"}],
						"shellCommand": "echo ${TEST_ENV_VAR}",
						"stdoutField": "spec.atFunction.shell.stdout"
					}`),
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"spec": {
									"atFunction": {
										"shell": {
											"stdout": "foo"
										}
									}
								},
								"status": {
									"atFunction": {
										"shell": {
											"stderr": ""
										}
									}
								}
							}`),
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := &Function{log: logging.NewNopLogger()}
			rsp, err := f.RunFunction(tc.args.ctx, tc.args.req)

			if diff := cmp.Diff(tc.want.rsp, rsp, protocmp.Transform()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}
