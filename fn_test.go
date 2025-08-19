package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
)

func TestRunFunction(t *testing.T) {

	type args struct {
		ctx context.Context
		req *fnv1.RunFunctionRequest
	}
	type want struct {
		rsp *fnv1.RunFunctionResponse
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
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  "invalid Function input: parameters: Required value: one of ShellCommand or ShellCommandField is required",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
		"ResponseIsEmptyShellCommand": {
			reason: "The Function should return a response when after a script is run",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": ""
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  "invalid Function input: parameters: Required value: one of ShellCommand or ShellCommandField is required",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
		"ResponseIsEcho": {
			reason: "The Function should write stdout to the specified field",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": "echo foo",
						"stdoutField": "spec.atFunction.shell.stdout"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1.State{
						Composite: &fnv1.Resource{
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
		"ResponseIsErrorWhenShellCommandNotFound": {
			reason: "The Function should write to the specified stderr field when the shellCommand is not found",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": "unknown-shell-command",
						"stdoutField": "spec.atFunction.shell.stdout",
						"stderrField": "spec.atFunction.shell.stderr"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  "shellCmd unknown-shell-command for  failed: exit status 127",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
		"ResponseIsEchoEnvVar": {
			reason: "The Function should accept and use environment variables",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
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
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1.State{
						Composite: &fnv1.Resource{
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
		"ResponseIsEchoShellEnvVarFieldPath": {
			reason: "The Function should accept and use environment variables from a fieldPath",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellEnvVars": [{"key": "TEST_ENV_VAR", "fieldRef":{"path": "spec.foo", "policy": "Required"}, "type": "FieldRef"}],
						"shellCommand": "echo ${TEST_ENV_VAR}",
						"stdoutField": "spec.atFunction.shell.stdout"
					}`),
					Observed: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"spec": {
									"foo": "bar"
								}
							}`),
						},
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"spec": {
									"atFunction": {
										"shell": {
											"stdout": "bar"
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
		"ResponseIsEchoEnvVarFieldRefDefaultValue": {
			reason: "The Function should accept and use environment variables from a default FieldRef ",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellEnvVars": [{"key": "TEST_ENV_VAR", "fieldRef":{"path": "spec.bad", "policy": "Optional", "defaultValue": "default"}, "type": "FieldRef"}],
						"shellCommand": "echo ${TEST_ENV_VAR}",
						"stdoutField": "spec.atFunction.shell.stdout"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"spec": {
									"atFunction": {
										"shell": {
											"stdout": "default"
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
		"ResponseIsErrorWhenBadShellEnvVarTypeIsProvided": {
			reason: "The Function should return an error when a bad shellEnVars type is provided",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellEnvVars": [{"key": "TEST_ENV_VAR", "value": "foo", "type": "bad"}],
						"shellCommand": "echo ${TEST_ENV_VAR}",
						"stdoutField": "spec.atFunction.shell.stdout"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  "shellEnvVars: unknown type bad for key TEST_ENV_VAR",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
		"ResponseIsErrorWhenShellEnvVarFieldPathIsEmpty": {
			reason: "The Function should return an error when a shellEnvVars fieldRef.path is empty",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellEnvVars": [{"key": "TEST_ENV_VAR", "fieldRef":{"policy": "Optional", "defaultValue": "default"}, "type": "FieldRef"}],
						"shellCommand": "echo ${TEST_ENV_VAR}",
						"stdoutField": "spec.atFunction.shell.stdout"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  "cannot process contents of fieldRef : path must be set",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
		"ResponseWithCustomCacheTTL": {
			reason: "The Function should set custom TTL when cacheTTL is specified",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": "echo test",
						"cacheTTL": "5m",
						"stdoutField": "spec.atFunction.shell.stdout"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(5 * time.Minute)},
					Desired: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"spec": {
									"atFunction": {
										"shell": {
											"stdout": "test"
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
		"ResponseWithInvalidCacheTTL": {
			reason: "The Function should return a fatal error when cacheTTL has invalid format",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": "echo test",
						"cacheTTL": "5x",
						"stdoutField": "spec.atFunction.shell.stdout"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  `cannot set cacheTTL: time: unknown unit "x" in duration "5x"`,
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
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
