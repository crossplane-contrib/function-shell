# Arbitrary UID Example

This example demonstrates how to run function-shell as an arbitrary user ID,
which is common in OpenShift and other restricted Kubernetes environments.

## Problem

Some Kubernetes environments force containers to run as arbitrary user IDs
that don't exist in `/etc/passwd`. This causes tools like AWS CLI, git, and
ssh to fail because they can't look up the current user.

## Solution

The function-shell image includes `nss-wrapper` support. When enabled via
a `DeploymentRuntimeConfig`, the entrypoint script creates fake passwd/group
entries for the running UID/GID, allowing tools that require user lookup to
function correctly.

## Files

- `functions.yaml` - Function and DeploymentRuntimeConfig with nss-wrapper enabled
- `composition.yaml` - Example composition that prints user info (HOME, UID, GID, whoami)
- `xr.yaml` - Example composite resource

## Testing Locally

Note: Local testing with `crossplane beta render` runs the function on your host machine
without the container's user configuration. The nss-wrapper functionality and arbitrary UID
handling can only be fully tested in a Kubernetes cluster (see "Deploying to a Cluster" below).

If you want to test the function logic locally:

1. Start the function in development mode:

   ```shell
   go run . --insecure --debug
   ```

2. In another terminal, render the example (this will use your host user, not UID 2000):

   ```shell
   crossplane beta render \
       example/arbitrary-uid/xr.yaml \
       example/arbitrary-uid/composition.yaml \
       example/arbitrary-uid/functions.yaml
   ```

## Deploying to a Cluster

1. Install the function with the DeploymentRuntimeConfig:

   ```shell
   kubectl apply -f example/arbitrary-uid/functions.yaml
   ```

2. Apply the CompositeResourceDefinition and Composition:

   ```shell
   kubectl apply -f example/arbitrary-uid/definition.yaml
   kubectl apply -f example/arbitrary-uid/composition.yaml
   ```

3. Create a composite resource to trigger the composition:

   ```shell
   kubectl apply -f example/arbitrary-uid/xr.yaml
   ```

4. Check the composite resource status:

   ```shell
   kubectl get arbitraryuid example-arbitrary-uid -o yaml
   ```

   The output should show the function successfully ran with the arbitrary UID:

   ```yaml
   apiVersion: example.crossplane.io/v1
   kind: ArbitraryUID
   metadata:
     name: example-arbitrary-uid
   spec:
     crossplane:
       compositionRef:
         name: arbitrary-uid-example
   status:
     atFunction:
       shell:
         stderr: ""
         stdout: |-
           HOME=/tmp/home
           uid=2000(runner) gid=2000(runner) groups=2000(runner)
           whoami=runner
     conditions:
     - lastTransitionTime: "2026-07-27T17:45:59Z"
       reason: ReconcileSuccess
       status: "True"
       type: Synced
     - lastTransitionTime: "2026-07-27T17:45:59Z"
       reason: Available
       status: "True"
       type: Ready
   ```
