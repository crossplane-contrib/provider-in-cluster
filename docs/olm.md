# OLM

We currently support a resource which wraps arbitrary OLM packages.

## Vanilla Kubernetes

For a fresh Kubernetes cluster (e.g., `kind`), you can install the OLM by running the following command: 
```bash
curl -sL https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.17.0/install.sh | bash -s v0.17.0
```

You can see installable operators by running the following command: 
```bash
kubectl get packagemanifests -n olm
```
Which displays all the available packagemanifests, each of which can correspond to one or more subscription.

Prior to creating a though subscription, you will need to create an operator-group, there are plans to include this automatic reconsilation for operator groups in the lifecycle for Operator resources in the future.

## Openshift

For an Openshift cluster, the OLM will be installed already, and you can see available operators by running the following command: 
```bash
kubectl get packagemanifests -n openshift-marketplace
```
By default, there should be a valid operator group in the default namespace.

## Example

To use the Operator resource, you need to configure a few key fields, namely:

- `operatorName` - Which is the internal name of the operator, note that this can differ from the name exposed by the call to list all packagemanifests.
- `catalogSource` - The specific catalog resource that exposes the operator 
- `catalogSourceNamespace` - The namespace in which that catalog resource resides
- `channel` - The channel that should be created, typically you will want to use the stable channel, however some operators exposes different versions (e.g., main, alpha, etc.)

Examples of the operator resource can found under the [examples/operator](../examples/operator) directory.
