# Terraform

A simple module to set up some requirements for a Taskcluster deployment.

To use this module, you must have configured the following terraform providers
in your module:

- aws
- azurerm
- rabbitmq
- [k8s](https://github.com/ericchiang/terraform-provider-k8s)
- [json-e](https://github.com/taskcluster/terraform-provider-jsone)

You will also need to `kubectl config use-context <the cluster you wish to deploy to>`
in the shell you are applying this from.

## Requirements not managed here

- A kubernetes cluster
- An [nginx ingress controller](https://kubernetes.github.io/ingress-nginx/deploy/) in said cluster
- A TLS secret for the rootUrl hostname in the same Kubernetes namespace as the controller
- A rabbitmq cluster with the RabbitMQ management plugin enabled
- An SES email address set up in AWS. This cannot be created automatically by Terraform.

## Usage

First include this module in your terraform.

```hcl
module "taskcluster" {
  source                    = "github.com/taskcluster/taskcluster-terraform"
  prefix                    = "tc"
  root_url                  = "tc.example.com"
  root_url_tls_secret       = "tc-example-com-tls-secret"
  azure_region              = "${var.azure_region}"
  rabbitmq_hostname         = "${var.rabbitmq_hostname}"
  rabbitmq_vhost            = "${var.rabbitmq_vhost}"
}
```

Before you apply, you should `kubectl config use-context <the cluster you wish to deploy to>`

The root accessToken is also available as output.

## TODO

- [ ] Ensure documentation for all variables and outputs
