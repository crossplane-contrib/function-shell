# function-shell

This Crossplane composition [function][functions] is written in [go][go]
following this [function guide][function guide]. It runs in a [docker][docker]
container. The [package docs][package docs] are a useful reference when
writing functions.

This is the `v1alpha1` version of `function-shell`.
Once [this pull request](https://github.com/crossplane/crossplane/pull/5543)
to introduce how to support passing credentials
to composition functions has been merged, the current functionality
for how to pass secrets in `function-shell`
is expected to follow the above pattern.

The `function-shell` accepts commands to run in a shell and it
returns the output to specified fields. It accepts the following parameters:

- `shellEnvVarsSecretRef` - referencing environment variables in a
Kubernetes secret. `shellEnvVarsSecretRef` requires a `name`, a
`namespace` and a `key` for the secret. Inside of it, the shell
expects a JSON structure with key value environment variables. Example:

```json
{
    "ENV_FOO": "foo value",
    "ENV_BAR": "bar value"
}
```

- `shellEnvVars` - an array of environment variables with a
`key` and `value` each.
- `shellCommand` - a shell command line that can contain pipes
and redirects and calling multiple programs.
- `shellCommandField` - a reference to a field that contains
the shell command line that should be run.
- `stdoutField` - the path to the field where the shell
standard output should be written.
- `stderrField` - the path to the field where the shell
standard error output should be written.

## Practical Example: Obtain Dashboard Ids from Datadog

The composition calls the `function-shell` instructing it to obtain dashboard
ids from a [Datadog](https://www.datadoghq.com/) account.
For this, the composition specifies the location
of a Kubernetes secret where the `DATADOG_API_KEY` and `DATADOG_APP_KEY`
environment variable values are stored. The Datadog API endpoint is passed
in a clear text `DATADOG_API_URL` environment variable. The shell command
uses a `curl` to the endpoint with a header that contains the access
credentials. The command output is piped into
[jq](https://jqlang.github.io/jq/) and filtered for the ids.

The `function-shell` writes the dashboard ids to the
specified output status field, and any output that went
to stderr into the specified stderr status field.

The composition is for illustration purposes only. When using the
`function-shell` in your own compositions, you may want to patch function input
from claim and other composition field values.

Note: `function-shell` requires permissions in form of a `rolebinding` to
read secrets and perform other actions that may be prohibited by default. Below
is a `clusterrolebinding` that will work, but you should exercise appropriate
caution when setting function permissions.

```shell
#!/bin/bash
NS="crossplane-system" # Replace with the namespace you use, e.g. upbound-system
SA=$(kubectl -n ${NS} get sa -o name | grep function-shell | sed -e 's|serviceaccount\/|${NS}:|g')
kubectl create clusterrolebinding function-shell-admin-binding \
    --clusterrole cluster-admin \
    --serviceaccount="${SA}"
```

The composition reads a datadog secret that looks like below.
Replace `YOUR_API_KEY` and `YOUR_APP_KEY` with your respective keys.

```json
{
    "DATADOG_API_KEY": "YOUR_API_KEY",
    "DATADOG_APP_KEY": "YOIR_APP_KEY"
}
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

```yaml
---
apiVersion: upbound.io/v1alpha1
kind: Shell
metadata:
  name: shell-1
spec: {}
```

The API definition is as follows. Note that the API contains status fields that
are populated by `function-shell`.

```yaml
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

```shell
crossplane beta trace shell.upbound.io/shell-1
NAME                      SYNCED   READY   STATUS
Shell/shell-1 (default)   True     True    Available
└─ XShell/shell-1-ttfbh   True     True    Available
```

The `XShell/shell-1-ttfbh` yaml output looks as per below. Notice the dashboard
ids in the `status.atFunction.shell.stdout` field, and the `curl` stderr output
in the `status.atFunction.shell.stderr` field.

```yaml
apiVersion: upbound.io/v1alpha1
kind: XShell
metadata:
  creationTimestamp: "2024-04-11T02:31:54Z"
  finalizers:
  - composite.apiextensions.crossplane.io
  generateName: shell-1-
  generation: 17
  labels:
    crossplane.io/claim-name: shell-1
    crossplane.io/claim-namespace: default
    crossplane.io/composite: shell-1-wjjs4
  name: shell-1-wjjs4
  resourceVersion: "2577566"
  uid: 77d24f9f-96db-4758-9155-9364ad227d0a
spec:
  claimRef:
    apiVersion: upbound.io/v1alpha1
    kind: Shell
    name: shell-1
    namespace: default
  compositionRef:
    name: shell.upbound.io
  compositionRevisionRef:
    name: shell.upbound.io-2403237
  compositionUpdatePolicy: Automatic
  resourceRefs: []
status:
  atFunction:
    shell:
      stderr: "% Total    % Received % Xferd  Average Speed   Time    Time     Time
        \ Current\n                                 Dload  Upload   Total   Spent
        \   Left  Speed\n\r  0     0    0     0    0     0      0      0 --:--:--
        --:--:-- --:--:--     0\r100  4255  100  4255    0     0   8862      0 --:--:--
        --:--:-- --:--:--  8864"
      stdout: |-
        "vn4-agn-ftd"
        "9pt-bhb-uwj"
        "6su-nff-222"
        "sm3-cxs-q98"
        "ssx-sci-uvi"
        "3fd-h4e-7w6"
        "qth-94z-ip5"
        Python
  conditions:
  - lastTransitionTime: "2024-04-11T02:53:01Z"
    reason: ReconcileSuccess
    status: "True"
    type: Synced
  - lastTransitionTime: "2024-04-11T02:31:55Z"
    reason: Available
    status: "True"
    type: Ready
```

## Development and test

Crossplane has a [cli][cli] with useful commands for building packages.

### Function code generation

```shell
go generate ./...
```

### Build the function's runtime image - see Dockerfile

```shell
docker build . --tag=runtime
```

### Render example function output

In Terminal 1

```shell
go run . --insecure --debug
```

In Terminal 2

```shell
crossplane beta render \
    example/out-of-cluster/xr.yaml \
    example/out-of-cluster/composition.yaml \
    example/out-of-cluster/functions.yaml
```

### Lint code

```shell
golangci-lint run
```

### Run tests

```shell
go test -v -cover .
```

### Docker build amd64 image

```shell
docker build . --quiet --platform=linux/amd64 --tag runtime-amd64
```

### Docker build arm64 image

```shell
docker build . --quiet --platform=linux/arm64 --tag runtime-arm64
```

### Crossplane build amd64 package

```shell
crossplane xpkg build \
    --package-root=package \
    --embed-runtime-image=runtime-amd64 \
    --package-file=function-amd64.xpkg
```

### Crossplane build arm64 package

```shell
crossplane xpkg build \
    --package-root=package \
    --embed-runtime-image=runtime-arm64 \
    --package-file=function-arm64.xpkg
```

## References

[functions]: https://docs.crossplane.io/latest/concepts/composition-functions
[go]: https://go.dev
[function guide]: https://docs.crossplane.io/knowledge-base/guides/write-a-composition-function-in-go
[package docs]: https://pkg.go.dev/github.com/crossplane/function-sdk-go
[docker]: https://www.docker.com
[cli]: https://docs.crossplane.io/latest/cli
