# provider-in-cluster

## Note this Provider in early alpha, breaking changes may occur!

The Provider-In-Cluster is designed to provision and configure resources in an arbitrary Kubernetes cluster. These resources run inside of the cluster, and are also managed by the cluster.

## How to Run

### Prerequisites

1. Crossplane Operator - instructions to install this operator can be found on the [Crossplane Docs](https://crossplane.io/docs/v0.14/getting-started/install-configure.html)
2. Crossplane CLI - the [Crossplane CLI](https://crossplane.io/docs/v0.14/getting-started/install-configure.html#install-crossplane-cli) will make installing providers very simple
3. Kubernetes Cluster - You will need a Kubernetes cluster :)

## Installing

You can install the most recent version of this Provider by running the following command:
```bash
kubectl crossplane install provider crossplane/provider-in-cluster:master
```

After installing, you will need a ProviderConfig to provide appropriate credentials for the operator. An example of how to achieve this can be found in the [examples directory](examples/).

## Getting Started

There are examples of the supported resources under the [examples](examples/) folder. This includes the providerConfig, postgres and operator resources.

To run the provider-in-cluster, you first need to set up the ProviderConfig, which expects a secret which contains a valid Kubeconfig to access the cluster. 
There is a [utility script](examples/kubeconfig.sh) to create the secret which expects a path to a Kubeconfig. After, you can create the 
[provider-config](examples/provider-config.yaml). Now, you should be able to create any of the other resources made available by the provider-in-cluster.

### OLM

Under the OLM package we support the creation of arbitrary operator SDK operators. This uses the existing subscription mechanism in the OLM (Operator Lifecycle Manager).

More detailed documentation can be found in the [OLM docs](docs/olm.md).

### Database

Under the database package we currently support provisioning instances of Postgres.

More documentation on this can be found in the [database docs](docs/database.md).

## Planned support

- Redis in cluster
- Object storage
