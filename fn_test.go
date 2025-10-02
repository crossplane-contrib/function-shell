package main

import (
	"context"
	"regexp"
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
		ctx      context.Context
		req      *fnv1.RunFunctionRequest
		useRegex bool // regex match on message due to differing error messages between shells
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
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
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
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
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
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
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
		"ResponseIsErrorIfInvalidShellCommand": {
			reason: "The function should write to the specified stderr when the shell command is invalid",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": "set -euo pìpefail",
						"stdoutField": "status.atFunction.shell.stdout",
						"stderrField": "status.atFunction.shell.stderr"
                    }`),
				},
				useRegex: true,
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"status": {
									"atFunction": {
										"shell": {
                                            "stdout": "",
											"stderr": "/bin/sh: .*set: .*pìpefail"
										}
									}
								}
							}`),
						},
					},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  "shellCmd \"set -euo pìpefail\" for \"\" failed with : exit status 2",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
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
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": "unknown-shell-command",
						"stdoutField": "status.atFunction.shell.stdout",
						"stderrField": "status.atFunction.shell.stderr"
					}`),
				},
				useRegex: true,
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"status": {
									"atFunction": {
										"shell": {
                                            "stdout": "",
											"stderr": "/bin/sh: .*unknown-shell-command.*"
										}
									}
								}
							}`),
						},
					},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  "shellCmd unknown-shell-command for failed: exit status 127",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
		"ResponseIsFailingCommandWithOutput": {
			reason: "The Function should capture both stdout and stderr when a command fails but produces output",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
						"kind": "Parameters",
						"shellCommand": "echo 'success output'; echo 'error output' >&2; exit 1",
						"stdoutField": "status.atFunction.shell.stdout",
						"stderrField": "status.atFunction.shell.stderr"
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
								"status": {
									"atFunction": {
										"shell": {
											"stdout": "success output",
											"stderr": "error output"
										}
									}
								}
							}`),
						},
					},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  "shellCmd \"echo 'success output'; echo 'error output' >&2; exit 1\" for \"\" failed with error output",
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
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
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
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
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
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
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
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
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
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
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
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
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
						"apiVersion": "shell.fn.crossplane.io/v1alpha1",
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

			var cmpOpts []cmp.Option
			cmpOpts = append(cmpOpts, protocmp.Transform(), protocmp.IgnoreFields(&fnv1.Result{}, "message"))

			if tc.args.useRegex {
				cmpOpts = append(cmpOpts, cmp.Comparer(func(expected, actual string) bool {
					// If expected looks like a regex pattern, use regex matching
					if regexp.MustCompile(`^.*\.\*.*$`).MatchString(expected) {
						matched, _ := regexp.MatchString(expected, actual)
						return matched
					}
					return expected == actual
				}))
			}

			if diff := cmp.Diff(tc.want.rsp, rsp, cmpOpts...); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}
