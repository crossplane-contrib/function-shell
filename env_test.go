package main

import (
	"testing"

	"github.com/crossplane-contrib/function-shell/input/v1alpha1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestFromValueRef(t *testing.T) {

	type args struct {
		req  *fnv1.RunFunctionRequest
		path string
	}

	type want struct {
		result string
		err    error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"FromCompositeValid": {
			reason: "If composite path is valid, it should be returned.",
			args: args{
				req: &fnv1.RunFunctionRequest{
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
				path: "spec.foo",
			},
			want: want{
				result: "bar",
				err:    nil,
			},
		},
		"FromContextValid": {
			reason: "If composite path is valid, it should be returned.",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Context: resource.MustStructJSON(`{
						"apiextensions.crossplane.io/foo": {
							"bar": "baz"
							}
					}`),
				},
				path: "context[apiextensions.crossplane.io/foo].bar",
			},
			want: want{
				result: "baz",
				err:    nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := fromValueRef(tc.args.req, tc.args.path)

			if diff := cmp.Diff(tc.want.result, result, protocmp.Transform()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}

}

func TestFromFieldRef(t *testing.T) {

	type args struct {
		req      *fnv1.RunFunctionRequest
		fieldRef v1alpha1.FieldRef
	}

	type want struct {
		result string
		err    error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"FromCompositeValid": {
			reason: "If composite path is valid, it should be returned.",
			args: args{
				req: &fnv1.RunFunctionRequest{
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
				fieldRef: v1alpha1.FieldRef{
					Path: "spec.foo",
				},
			},
			want: want{
				result: "bar",
				err:    nil,
			},
		},
		"FromCompositeMissingError": {
			reason: "If composite path is invalid and Policy is required, return an error",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Observed: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"spec": {
									"bad": "bar"
								}
							}`),
						},
					},
				},
				fieldRef: v1alpha1.FieldRef{
					Path:   "spec.foo",
					Policy: v1alpha1.FieldRefPolicyRequired,
				},
			},
			want: want{
				result: "",
				err:    errors.New("cannot get observed composite value at spec.foo: spec.foo: no such field"),
			},
		},
		"FromCompositeMissingErrorDefaultPolicy": {
			reason: "If composite path is invalid and Policy is not set, return an error",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Observed: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"spec": {
									"bad": "bar"
								}
							}`),
						},
					},
				},
				fieldRef: v1alpha1.FieldRef{
					Path: "spec.foo",
				},
			},
			want: want{
				result: "",
				err:    errors.New("cannot get observed composite value at spec.foo: spec.foo: no such field"),
			},
		},
		"FromCompositeMissingFieldOptionalPolicyDefaultValue": {
			reason: "If composite path is invalid and Policy is set to Optional, return DefaultValue",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Observed: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"spec": {
									"bad": "bar"
								}
							}`),
						},
					},
				},
				fieldRef: v1alpha1.FieldRef{
					DefaultValue: "default",
					Path:         "spec.foo",
					Policy:       v1alpha1.FieldRefPolicyOptional,
				},
			},
			want: want{
				result: "default",
				err:    nil,
			},
		},
		"FromCompositeMissingFieldOptionalPolicyNoDefaultValue": {
			reason: "If composite path is invalid, Policy is set to Optional, and no defaultValue return empty string",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Observed: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "",
								"kind": "",
								"spec": {
									"bad": "bar"
								}
							}`),
						},
					},
				},
				fieldRef: v1alpha1.FieldRef{
					Path:   "spec.foo",
					Policy: v1alpha1.FieldRefPolicyOptional,
				},
			},
			want: want{
				result: "",
				err:    nil,
			},
		},
		"FromContextValid": {
			reason: "If context path is valid, it should be returned.",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Context: resource.MustStructJSON(`{
						"apiextensions.crossplane.io/foo": {
							"bar": "baz"
							}
					}`),
				},
				fieldRef: v1alpha1.FieldRef{
					Path: "context[apiextensions.crossplane.io/foo].bar",
				},
			},
			want: want{
				result: "baz",
				err:    nil,
			},
		},
		"FromContextMissingError": {
			reason: "If context path is invalid and Policy is Required, return an error",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Context: resource.MustStructJSON(`{
						"apiextensions.crossplane.io/foo": {
							"bar": "baz"
							}
					}`),
				},
				fieldRef: v1alpha1.FieldRef{
					Path:   "context[apiextensions.crossplane.io/foo].bad",
					Policy: v1alpha1.FieldRefPolicyRequired,
				},
			},
			want: want{
				result: "",
				err:    errors.New("cannot get context value at bad: bad: no such field"),
			},
		},
		"FromContextMissingErrorDefaultPolicy": {
			reason: "If context path is invalid and no Policy is defined, return an error",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Context: resource.MustStructJSON(`{
						"apiextensions.crossplane.io/foo": {
							"bar": "baz"
							}
					}`),
				},
				fieldRef: v1alpha1.FieldRef{
					Path: "context[apiextensions.crossplane.io/foo].bad",
				},
			},
			want: want{
				result: "",
				err:    errors.New("cannot get context value at bad: bad: no such field"),
			},
		},
		"FromContextMissingFieldOptionalPolicyNoDefaultValue": {
			reason: "If context path is invalid, Policy is Optional, and no DefaultValue return empty string",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Context: resource.MustStructJSON(`{
						"apiextensions.crossplane.io/foo": {
							"bar": "baz"
							}
					}`),
				},
				fieldRef: v1alpha1.FieldRef{
					Path:   "context[apiextensions.crossplane.io/foo].bad",
					Policy: v1alpha1.FieldRefPolicyOptional,
				},
			},
			want: want{
				result: "",
				err:    nil,
			},
		},
		"FromContextMissingFieldOptionalPolicyDefaultValue": {
			reason: "If context path is invalid, Policy is Optional, return DefaultValue",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Context: resource.MustStructJSON(`{
						"apiextensions.crossplane.io/foo": {
							"bar": "baz"
							}
					}`),
				},
				fieldRef: v1alpha1.FieldRef{
					DefaultValue: "default",
					Path:         "context[apiextensions.crossplane.io/foo].bad",
					Policy:       v1alpha1.FieldRefPolicyOptional,
				},
			},
			want: want{
				result: "default",
				err:    nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := fromFieldRef(tc.args.req, tc.args.fieldRef)

			if diff := cmp.Diff(tc.want.result, result, protocmp.Transform()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", tc.reason, diff)
			}

			// deal with internal fieldpath errors that are generated by a private method
			if tc.want.err != nil && err != nil {
				if diff := cmp.Diff(tc.want.err.Error(), err.Error()); diff != "" {
					t.Errorf("%s\nf.RunFunction(...): -want err message, +got err message:\n%s", tc.reason, diff)
				}
			} else if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}

}
