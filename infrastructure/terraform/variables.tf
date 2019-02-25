variable "prefix" {
  type        = "string"
  description = "Short prefix applied to all cloud resources needed for a taskcluster cluster to function. This should be different for each deployment sharing a cloud account."
}

variable "azure_region" {
  type        = "string"
  description = "Region of azure storage"
}

variable "aws_region" {
  description = "The AWS region to deploy into (e.g. us-east-1)."
}

variable "kubernetes_namespace" {
  default     = "taskcluster"
  type        = "string"
  description = "Optional namespace to run services in"
}

variable "root_url" {
  type        = "string"
  description = "Taskcluster rootUrl"
}

variable "root_url_tls_secret" {
  type        = "string"
  description = "Name of the secret, in the same namespace as the Ingress controller, containing the TLS certificate for Taskcluster rootUrl"
}

variable "rabbitmq_hostname" {
  type        = "string"
  description = "rabbitmq hostname"
}

variable "rabbitmq_vhost" {
  type        = "string"
  description = "rabbitmq vhost name"
}

variable "notify_ses_arn" {
  type        = "string"
  description = "arn of an ses address. This must be manually set up in aws."
}

variable "disabled_services" {
  type        = "list"
  default     = []
  description = "List of services to disable i.e. [\"taskcluster-notify\"]"
}

variable "cluster_name" {
  type        = "string"
  description = "Human readable cluster name"
}

variable "irc_name" {
  type        = "string"
  description = "username for irc bot."
}

variable "irc_nick" {
  type        = "string"
  description = "nick for irc bot."
}

variable "irc_real_name" {
  type        = "string"
  description = "real name for irc bot."
}

variable "irc_server" {
  type        = "string"
  description = "server for irc bot."
}

variable "irc_port" {
  type        = "string"
  description = "port for irc bot."
}

variable "irc_password" {
  type        = "string"
  description = "password for irc bot."
}

variable "github_app_id" {
  type        = "string"
  description = "taskcluster-github app id."
}

variable "github_oauth_token" {
  type        = "string"
  description = "taskcluster-github app oauth token."
}

variable "github_private_pem" {
  type        = "string"
  description = "taskcluster-github private pem."
}

variable "github_webhook_secret" {
  type        = "string"
  description = "taskcluster-github webhook secret."
}

variable "audit_log_stream" {
  type        = "string"
  description = "kinesis stream for audit logs."
}

variable "gce_provider_gcp_project" {
  type        = "string"
  description = "Project in Google Cloud (used for gce_provider)."
}

variable "gce_provider_image_name" {
  type        = "string"
  description = "Image name to use for workers spawned by gce_provider."
}
