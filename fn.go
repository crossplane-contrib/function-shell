package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/crossplane-contrib/function-shell/input/v1alpha1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/keegancsmith/shell"
	"google.golang.org/protobuf/types/known/durationpb"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/response"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction runs the Function.
//
//gocognit:ignore
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	in := &v1alpha1.Parameters{}
	if err := request.GetInput(req, in); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function from input"))
		return rsp, nil
	}

	if in.CacheTTL != "" {
		dur, err := time.ParseDuration(in.CacheTTL)
		if err != nil {
			response.Fatal(rsp, errors.Wrapf(err, "cannot set cacheTTL"))
			return rsp, nil
		}
		rsp.Meta.Ttl = durationpb.New(dur)
	}

	oxr, err := request.GetObservedCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get observed composite resource from %T", req))
		return rsp, nil
	}

	// Our input is an opaque object nested in a Composition. Let's validate it
	if err := ValidateParameters(in, oxr); err != nil {
		response.Fatal(rsp, errors.Wrap(err, "invalid Function input"))
		return rsp, nil
	}

	log := f.log.WithValues(
		"oxr-version", oxr.Resource.GetAPIVersion(),
		"oxr-kind", oxr.Resource.GetKind(),
		"oxr-name", oxr.Resource.GetName(),
	)

	dxr, err := request.GetDesiredCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot get desired composite resource"))
		return rsp, nil
	}

	dxr.Resource.SetAPIVersion(oxr.Resource.GetAPIVersion())
	dxr.Resource.SetKind(oxr.Resource.GetKind())

	stdoutField := in.StdoutField
	if len(in.StdoutField) == 0 {
		stdoutField = "status.atFunction.shell.stdout"
	}
	stderrField := in.StderrField
	if len(in.StderrField) == 0 {
		stderrField = "status.atFunction.shell.stderr"
	}

	shellCmd := ""
	if len(in.ShellCommand) == 0 && len(in.ShellCommandField) == 0 {
		log.Info("no shell command in in.ShellCommand nor in.ShellCommandField")
		return rsp, nil
	}

	if len(in.ShellCommand) > 0 {
		shellCmd = in.ShellCommand
	}

	// Prefer shell cmd from field over direct function input
	if len(in.ShellCommandField) > 0 {
		shellCmd = in.ShellCommandField
	}

	shellEnvVars := make(map[string]string)
	for _, envVar := range in.ShellEnvVars {
		switch t := envVar.GetType(); t {
		case v1alpha1.ShellEnvVarTypeValue:
			shellEnvVars[envVar.Key] = envVar.Value
		case v1alpha1.ShellEnvVarTypeValueRef:
			envValue, err := fromValueRef(req, envVar.ValueRef)
			if err != nil {
				response.Fatal(rsp, errors.Wrapf(err, "cannot process contents of valueRef %s", envVar.ValueRef))
				return rsp, nil
			}
			shellEnvVars[envVar.Key] = envValue
		case v1alpha1.ShellEnvVarTypeFieldRef:
			envValue, err := fromFieldRef(req, *envVar.FieldRef)
			if err != nil {
				response.Fatal(rsp, errors.Wrapf(err, "cannot process contents of fieldRef %s", envVar.ValueRef))
				return rsp, nil
			}
			shellEnvVars[envVar.Key] = envValue
		default:
			response.Fatal(rsp, errors.Errorf("shellEnvVars: unknown type %s for key %s", t, envVar.Key))
			return rsp, nil
		}
	}

	if len(in.ShellEnvVarsRef.Keys) > 0 {
		shellEnvVars, err = addShellEnvVarsFromRef(in.ShellEnvVarsRef, shellEnvVars)
		if err != nil {
			response.Fatal(rsp, errors.Wrapf(err, "cannot process contents of shellEnvVarsRef %s", in.ShellEnvVarsRef.Name))
			return rsp, nil
		}
	}

	var exportCmds string
	// exportCmds = "export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin;"
	for k, v := range shellEnvVars {
		exportCmds = exportCmds + "export " + k + "=\"" + v + "\";"
	}

	log.Info(shellCmd)

	var stdout, stderr bytes.Buffer
	cmd := shell.Commandf(exportCmds + shellCmd)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmderr := cmd.Run()
	sout := strings.TrimSpace(stdout.String())
	serr := strings.TrimSpace(stderr.String())

	log.Debug(shellCmd, "stdout", sout, "stderr", serr)

	err = dxr.Resource.SetValue(stdoutField, sout)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set field %s to %s for %s", stdoutField, sout, oxr.Resource.GetKind()))
		return rsp, nil
	}

	err = dxr.Resource.SetValue(stderrField, serr)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set field %s to %s for %s", stderrField, serr, oxr.Resource.GetKind()))
	}
	if err := response.SetDesiredCompositeResource(rsp, dxr); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set desired composite resources from %T", req))
	}

	if cmderr != nil {
		exiterr := &exec.ExitError{}
		if errors.As(cmderr, &exiterr) {
			msg := fmt.Sprintf("shellCmd %q for %q failed with %s", shellCmd, oxr.Resource.GetKind(), exiterr.Stderr)
			response.Fatal(rsp, errors.Wrap(cmderr, msg))
		}
	}

	return rsp, nil
}
