package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/response"
	"github.com/giantswarm/function-shell-idp/input/v1alpha1"
	"github.com/keegancsmith/shell"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1beta1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1beta1.RunFunctionRequest) (*fnv1beta1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	in := &v1alpha1.Parameters{}
	if err := request.GetInput(req, in); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function from input"))
		return rsp, nil
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
		if envVar.ValueRef != "" {
			envValue, err := fromValueRef(req, envVar.ValueRef)
			if err != nil {
				response.Fatal(rsp, errors.Wrapf(err, "cannot process contents of valueRef %s", envVar.ValueRef))
				return rsp, nil
			}
			shellEnvVars[envVar.Key] = envValue
		} else {
			shellEnvVars[envVar.Key] = envVar.Value
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
		if exiterr, ok := cmderr.(*exec.ExitError); ok {
			msg := fmt.Sprintf("shellCmd %q for %q failed with %s", shellCmd, oxr.Resource.GetKind(), exiterr.Stderr)
			response.Fatal(rsp, errors.Wrap(cmderr, msg))
		}
	}

	return rsp, nil
}
