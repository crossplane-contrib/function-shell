# Example manifests

You can run your function locally and test it using `crossplane beta render`
with these example manifests.

```shell
# Run the function locally
$ go run . --insecure --debug
```

```shell
# Then, in another terminal, call it with these example manifests
$ crossplane beta render xr.yaml composition.yaml functions.yaml -r
---
apiVersion: example.crossplane.io/v1
kind: XR
metadata:
  name: example-xr
---
apiVersion: render.crossplane.io/v1beta1
kind: Result
message: I was run with input "Hello world"!
severity: SEVERITY_NORMAL
step: run-the-template
```


Make a Kubernetes secret for API and APP keys and credentials
in the form of a JSON file.
```
{
    "API_KEY": "...masked.value.here...",
    "APP_KEY": "...masked.value.here..."
}
```
For example, it can be created as follows:
```
kubectl -n upbound-system \
    create secret generic datadog-secret \
    --from-literal=credentials="${DATADOG_ENV_VARS_JSON}" \
    --dry-run=client \
    -o yaml|\
    kubectl apply -f -
```
