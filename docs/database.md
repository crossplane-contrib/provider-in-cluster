# Database

Currently under the database API group we only support provisioning instances PostgreSQL, but there are concrete plans to add support for other database applications. Note, in-cluster provisioning is also possible using the [Provider-Rook](https://github.com/crossplane/provider-rook), which support the Database services exposed by Rook.

## Example

Under the [examples/](../examples/database/) there is an example manifest for the postgres resource. This resource accepts a reference to a secret containing the password under the path `spec.forProvider.masterPasswordSecretRef`. The password secret is optional, if one is not provided, a password will be generated.

For storage, the resource accepts a StorageClass which will be used to access a PVC, if a valid PVC does not exist or cannot be provisioned, the creation of the Database instance will block.

When the resource is finished being provisioned, an output secret with the password, endpoint, database, port and username will be created.
