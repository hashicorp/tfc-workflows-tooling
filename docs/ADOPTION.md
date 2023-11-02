# Adopting TFCI

If you have any questions feel free to open an [issue](https://github.com/hashicorp/tfc-workflows-tooling/issues).

## Setup

Tfci currently supports the following CI/CD platforms:
* GitHub Actions
* GitLab Pipelines

However, Tfci can be instrumented for other platforms as well with the use of the [published Docker Container](https://hub.docker.com/r/hashicorp/tfci).

### Docker

The default artifact generated is a Docker container, hosted in the public Docker registry. `docker://hashicorp/tfci:{VERSION}`.

Leveraging the publicly distributed Docker image is the recommended approach for running Tfci on CI/CD platforms for non-human interactions.

View our [Usage documentation](./USAGE.md) to learn more our available commands, arguments, and configuration.

### How GitHub Actions uses tfci

[View all](https://github.com/hashicorp/tfc-workflows-github/tree/main/actions) available GitHub Actions that are built on top of tfci.

### [How GitLab Pipelines uses tfci](https://github.com/hashicorp/tfc-workflows-gitlab)

View the GitLab [Base-Template](https://github.com/hashicorp/tfc-workflows-gitlab/blob/main/Base.gitlab-ci.yml)

### Building a binary from source code

In scenarios where Docker is not available or feasible, you can build a binary directly from the source code.

Prerequisites:
* git installed
* Golang installed

Steps:
1. *[Clone the repository](https://github.com/hashicorp/tfc-workflows-tooling)*
1. Checkout from an available [release](https://github.com/hashicorp/tfc-workflows-tooling/releases).
1. Go cli to build a binary artifact `go build {flags}`

Example:
```bash
go build \
-ldflags "-X 'github.com/hashicorp/tfci/version.Version=$VERSION' \
-s \
-w \
-extldflags '-static'" \
-o /tfci \
.
```

## Workflow

### Terraform CLI vs. Terraform Cloud API

### Prescribed Workflows

#### Speculative Run

#### Plan/Apply Run
