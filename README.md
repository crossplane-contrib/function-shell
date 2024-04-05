# function-shell

This Crossplane composition function accepts commands to run in a shell and it
returns the output to specified fields. It accepts the following paramereters:
- `shellEnvVarsSecretRef` - referencing environment variables in a
  Kubernetes secret. shellEnvVarsSecretRef requires a `name`, a
`namespace` and a `key` for the secret. Inside of it, the shell
expects a JSON structure with key value environment variables. Example:

```
{
    "ENV_FOO": "foo value",
    "ENV_BAR": "bar value"
}
```
- 'shellEnvVars' - an array of environment variables with a <b>key</b> and <b>value</b>
  each.
- 'shellCommand' - a shell command line that can contain pipes and
  redirects and calling multiple programs.
- 'stdoutField' - the path to the field where the shell standard output
  should be written.
- 'stderrField' - the path to the field where the shell standard error
  output should be written.

## Practical Example: Obtain Dashboard Ids from Datadog

The composition calls the `function-shell` instructing it to obtain dashboard ids
from a Datadog account. For this, it specifies the location of a Kubernetes
secret where the `DATADOG_API_KEY` and `DATADOG_APP_KEY` environment variable values
are stored. The Datadog API endpoint is passed in a clear text environment
variable. The shell command uses a `curl` to the endpoint with a header that
contains the access credentials. The command output is piped into jq and
filtered for the ids.

The `function-shell` writes the dashboard ids to the specified output status field, and
any output that went to stderr into the specified stderr status field.

The composition is for illustration purposes only. When using the
`function-shell` in your own compositions, you may want to patch function input
from claim and other composition field values.

Note: `function-shell` has to receive permissions in form of a rolebinding to
read secrets and perform other actions that may be prohibited by default. Below
is a `clusterrolebinding` that will work, but you should exercise appropriate caution when
setting function permissions.

```
#!/bin/bash
SA=$(kubectl -n upbound-system get sa -o name | grep function-shell | sed -e 's|serviceaccount\/|upbound-system:|g')
kubectl create clusterrolebinding function-shell-admin-binding --clusterrole cluster-admin --serviceaccount="${SA}"
```

```yaml
---
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: shell.upbound.io
spec:
  compositeTypeRef:
    apiVersion: upbound.io/v1alpha1
    kind: XShell
  mode: Pipeline
  pipeline:
    - step: shell
      functionRef:
        name: function-shell
      input:
        apiVersion: shell.fn.crossplane.io/v1beta1
        kind: Parameters
        shellEnvVarsSecretRef:
          name: datadog-secret
          namespace: upbound-system
          key: credentials
        shellEnvVars:
          - key: DATADOG_API_URL
            value: "https://api.datadoghq.com/api/v1/dashboard"
        shellCommand: |
          curl -X GET "${DATADOG_API_URL}" \
            -H "Accept: application/json" \
            -H "DD-API-KEY: ${DATADOG_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DATADOG_APP_KEY}"|\
             jq '.dashboards[] .id';
        stdoutField: status.atFunction.shell.stdout
        stderrField: status.atFunction.shell.stderr
```

The composition is called through the following `claim`.

```
---
apiVersion: upbound.io/v1alpha1
kind: Shell
metadata:
  name: shell-1
spec: {}
```

The API definition is as follows. Note that the API contains status fields that
are populated by `function-shell`.

```
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xshells.upbound.io
spec:
  group: upbound.io
  names:
    kind: XShell
    plural: xshells
  claimNames:
    kind: Shell
    plural: shells
  defaultCompositionRef:
    name: shell.upbound.io
  versions:
    - name: v1alpha1
      served: true
      referenceable: true
      schema:
        openAPIV3Schema:
          properties:
            spec:
              properties:
                cmd:
                  type: string
            status:
              properties:
                atFunction:
                  type: object
                  x-kubernetes-preserve-unknown-fields: true
```

The `crossplane beta trace` output after applying the in-cluster
shell-claim.yaml is as follows:
```
cbt shell.upbound.io/shell-1
NAME                      SYNCED   READY   STATUS
Shell/shell-1 (default)   True     True    Available
└─ XShell/shell-1-ttfbh   True     True    Available
```

The `XShell/shell-1-ttfbh` yaml output looks as per below. Notice the dashboard
ids in the `status.atFunction.shell.stdout` field, and the `curl` stderr output
in the `status.atFunction.shell.stderr` field.

```
apiVersion: upbound.io/v1alpha1
kind: XShell
metadata:
  creationTimestamp: "2024-04-05T04:47:03Z"
  finalizers:
  - composite.apiextensions.crossplane.io
  generateName: shell-1-
  generation: 3
  labels:
    crossplane.io/claim-name: shell-1
    crossplane.io/claim-namespace: default
    crossplane.io/composite: shell-1-ttfbh
  name: shell-1-ttfbh
  resourceVersion: "2181275"
  uid: 9ebda770-bab3-4822-bd36-739cea5cd35e
spec:
  claimRef:
    apiVersion: upbound.io/v1alpha1
    kind: Shell
    name: shell-1
    namespace: default
  compositionRef:
    name: shell.upbound.io
  compositionRevisionRef:
    name: shell.upbound.io-ed28247
  compositionUpdatePolicy: Automatic
  resourceRefs: []
status:
  atFunction:
    shell:
      stderr: "% Total    % Received % Xferd  Average Speed   Time    Time     Time
        \ Current\n                                 Dload  Upload   Total   Spent
        \   Left  Speed\n\r  0     0    0     0    0     0      0      0 --:--:--
        --:--:-- --:--:--     0\r  0     0    0     0    0     0      0      0 --:--:--
        --:--:-- --:--:--     0\r100  4255  100  4255    0     0   9081      0 --:--:--
        --:--:-- --:--:--  9072"
      stdout: |-
        "vn4-agn-ftd"
        "9pt-bhb-uwj"
        "6su-nff-222"
        "sm3-cxs-q98"
        "ssx-sci-uvi"
        "3fd-h4e-7w6"
        "qth-94z-ip5"
  conditions:
  - lastTransitionTime: "2024-04-05T04:47:29Z"
    reason: ReconcileSuccess
    status: "True"
    type: Synced
  - lastTransitionTime: "2024-04-05T04:47:29Z"
    reason: Available
    status: "True"
    type: Ready
```

# Run code generation - see input/generate.go
$ go generate ./...

# Run tests - see fn_test.go
$ go test ./...

# Build the function's runtime image - see Dockerfile
$ docker build . --tag=runtime

# Build a function package - see package/crossplane.yaml
$ crossplane xpkg build -f package --embed-runtime-image=runtime
```

[functions]: https://docs.crossplane.io/latest/concepts/composition-functions
[go]: https://go.dev
[function guide]: https://docs.crossplane.io/knowledge-base/guides/write-a-composition-function-in-go
[package docs]: https://pkg.go.dev/github.com/crossplane/function-sdk-go
[docker]: https://www.docker.com
[cli]: https://docs.crossplane.io/latest/cli
