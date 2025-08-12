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
is expected to be enhanced with the above pattern.

The `function-shell` accepts commands to run in a shell and it
returns the output to specified fields. It accepts the following parameters:

- `shellEnvVarsRef` - referencing environment variables in the
function-shell Kubernetes pod that were loaded through a
`deploymentRuntimeConfig`. The file MUST be in `JSON` format.
It can be a Kubernetes secret. `shellEnvVarsRef` requires a `name`
for the pod environment variable, and `keys` for the keys Inside
of the JSON formatted pod environment variable that have associated
values.

Example secret:

```json
{
    "ENV_FOO": "foo value",
    "ENV_BAR": "bar value"
}
```

Example `deploymentRuntimeConfig`:

```yaml
---
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: function-shell
spec:
  deploymentTemplate:
    spec:
      selector: {}
      replicas: 1
      template:
        spec:
          containers:
            - name: package-runtime
              args:
                - --debug
              env:
                - name: DATADOG_SECRET
                  valueFrom:
                    secretKeyRef:
                      key: credentials
                      name: datadog-secret
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

## Caching Function Outputs

In Crossplane 1.20.0 and 2.0.0, Function Response Caching was added
as an alpha feature. Crossplane will cache the results of a function invocation
until a Time-To-Live (TTL) has been exceeded. This can significantly reduce
the number of times the function is called.

To enable Function Response Caching, update the crossplane deployment by adding `--enable-function-response-cache` to the `args` of the Crossplane deployment.

Next, set the `cacheTTL`, using a time duration like `90s`, `5m`, or `4h30m`:

```yaml
input:
  apiVersion: shell.fn.crossplane.io/v1alpha1
  kind: Parameters
  cacheTTL: 5m
  shellEnvVars:
    - key: ECHO
      value: "SGVsbG8gZnJvbSBzaGVsbAo="
  shellCommand: |
      echo ${ECHO}|base64 -d|sed s/^h/H/
  stdoutField: status.atFunction.shell.stdout
  stderrField: status.atFunction.shell.stderr
```

See the echo [composition.yaml](example/echo/composition.yaml) for an example.

## Examples

This repository includes the following examples:

- echo
- datadog-dashboard-ids
- ip-addr-validation

## Example: Obtain Dashboard Ids from Datadog

The composition calls the `function-shell` instructing it to obtain dashboard
ids from a [Datadog](https://www.datadoghq.com/) account.
For this, the composition specifies the name of a Kubernetes
pod environment variable called `DATADOG_SECRET`. This environment
variable was populated with the `JSON` of a Kubernetes datadog-secret
through a deploymentRuntimeConfig. The `JSON` includes the
`DATADOG_API_KEY` and `DATADOG_APP_KEY`
keys and their values. The Datadog API endpoint is passed
in a clear text `DATADOG_API_URL` environment variable. The shell command
uses a `curl` to the endpoint with a header that contains the access
credentials. The command output is piped into
[jq](https://jqlang.github.io/jq/) and filtered for the ids.

The `function-shell` writes the dashboard ids to the
specified output status field, and any output that went
to stderr into the specified stderr status field.

The composition is for illustration purposes only. When using the
`function-shell` in your own compositions,
you may want to patch function input
from claim and other composition field values.

The `deploymentRuntimeConfig` reads a datadog secret
that looks like below.
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
        # When installed through a package manager, use
        # name: crossplane-contrib-function-shell
        name: function-shell
      input:
        apiVersion: shell.fn.crossplane.io/v1beta1
        kind: Parameters
        # Load shellEnvVarsRef from a Kubernetes secret
        # through a deploymentRuntimeConfig into the
        # function-shell pod.
        shellEnvVarsRef:
          name: DATADOG_SECRET
          keys:
            - DATADOG_API_KEY
            - DATADOG_APP_KEY
        shellEnvVars:
          - key: DATADOG_API_URL
            value: "https://api.datadoghq.com/api/v1/dashboard"
        shellCommand: |
          curl -X GET "${DATADOG_API_URL}" \
            -H "Accept: application/json" \
            -H "DD-API-KEY: ${DATADOG_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DATADOG_APP_KEY}"|jq '.dashboards[] .id'
        stdoutField: status.atFunction.shell.stdout
        stderrField: status.atFunction.shell.stderr
```

The composition is selected through the following `XR`.

```yaml
---
apiVersion: upbound.io/v1alpha1
kind: Shell
metadata:
  name: shell-1
spec: {}
```

The API definition is as follows. Note that the API
contains status fields that are populated by `function-shell`.

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
crossplane beta trace shell.upbound.io/datadog-dashboard-ids
NAME                                    SYNCED   READY   STATUS
Shell/datadog-dashboard-ids (default)   True     True    Available
└─ XShell/datadog-dashboard-ids-cbb6x   True     True    Available
```

The `XShell/shell-1-ttfbh` yaml output looks as per below. Notice the dashboard
ids in the `status.atFunction.shell.stdout` field, and the `curl` stderr output
in the `status.atFunction.shell.stderr` field.

```yaml
kubectl get XShell/datadog-dashboard-ids-cbb6x -o yaml
apiVersion: upbound.io/v1alpha1
kind: XShell
metadata:
  creationTimestamp: "2024-04-24T04:15:53Z"
  finalizers:
  - composite.apiextensions.crossplane.io
  generateName: datadog-dashboard-ids-
  generation: 6
  labels:
    crossplane.io/claim-name: datadog-dashboard-ids
    crossplane.io/claim-namespace: default
    crossplane.io/composite: datadog-dashboard-ids-cbb6x
  name: datadog-dashboard-ids-cbb6x
  resourceVersion: "167413"
  uid: 601d3f66-80df-4f1a-8917-533ea05255cc
spec:
  claimRef:
    apiVersion: upbound.io/v1alpha1
    kind: Shell
    name: datadog-dashboard-ids
    namespace: default
  compositionRef:
    name: shell.upbound.io
  compositionRevisionRef:
    name: shell.upbound.io-e981893
  compositionUpdatePolicy: Automatic
  resourceRefs: []
status:
  atFunction:
    shell:
      stderr: "% Total    % Received % Xferd  Average Speed   Time    Time     Time
        \ Current\n                                 Dload  Upload   Total   Spent
        \   Left  Speed\n\r  0     0    0     0    0     0      0      0 --:--:--
        --:--:-- --:--:--     0\r100  4255  100  4255    0     0  10361      0 --:--:--
        --:--:-- --:--:-- 10378"
      stdout: |-
        "vn4-agn-ftd"
        "9pt-bhb-uwj"
        "6su-nff-222"
        "sm3-cxs-q98"
        "ssx-sci-uvi"
        "3fd-h4e-7w6"
        "qth-94z-ip5"
  conditions:
  - lastTransitionTime: "2024-04-24T04:20:09Z"
    reason: ReconcileSuccess
    status: "True"
    type: Synced
  - lastTransitionTime: "2024-04-24T04:15:54Z"
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
