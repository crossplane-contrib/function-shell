package main

import (
	"testing"

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
