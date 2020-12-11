# provider-in-cluster

### Note this Provider in early alpha, breaking changes may occur!

## How to Run

### Prerequisites
1. Crossplane Operator - instructions to install this operator can be found on the [Crossplane Docs](https://crossplane.io/docs/v0.14/getting-started/install-configure.html)
2. Crossplane CLI - the [Crossplane CLI](https://crossplane.io/docs/v0.14/getting-started/install-configure.html#install-crossplane-cli) will make installing providers very simple
3. Kubernetes Cluster - You will need a Kubernetes cluster :)

### Installing

You can install the most recent version of this Provider by running the following command:
`kubectl crossplane install provider crossplane/provider-in-cluster:master`

After installing, you will need a ProviderConfig to provide appropriate credentials for the operator. An example of how to achieve this can be found in the [examples directory](examples/).

## OLM

We currently support a resource to wrap arbitrary OLM packages. 

### Vanilla Kubernetes

For a fresh Kubernetes cluster (e.g., `kind`), you can install the OLM by running the following command: `curl -sL https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.17.0/install.sh | bash -s v0.17.0`

You can see available operators by running the following command: `kubectl get packagemanifests -n olm`. 

Prior to creating a subscription, you will need to create an operator-group, there are plans to include this lifecycle for Operator resources in the future.

### Openshift

For an Openshift cluster, the OLM will be installed already, and you can see available operators by running the following command: `kubectl get packagemanifests -n openshift-marketplace`. 


### Example

For using the Operator resource, you can see an example under the [examples/operator](examples/operator) directory.

## Database

We currently have support for Postgres in-cluster, this uses a PVC with a provided storage class. 

## Planned support

- Redis in cluster
- Object storage
