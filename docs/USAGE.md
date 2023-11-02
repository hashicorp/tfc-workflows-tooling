# Usage

## Available Commands

* `upload`: Creates and uploads configuration files for a given workspace
* `run show`: Returns run details for the provided Terraform Cloud Run ID.
* `run create`: Performs a new plan run in Terraform Cloud, using a configuration version and the workspace's current variables.
* `run apply`: Applies a run that is paused waiting for confirmation after a plan.
* `run discard`: Skips any remaining work on runs that are paused waiting for confirmation or priority.
* `run cancel`: Interrupts a run that is currently planning or applying.
* `plan output`: Returns the plan details for the provided Plan ID.
* `workspace output list`: Returns a list of workspace outputs.

## Pulling Image from Dockerhub

Pulling the latest version
```sh
docker pull hashicorp/tfci:latest
```

Pulling a specific version

```sh
docker pull hashicorp/tfci:v1.0.4
```

## Building the Image Locally

*Requires cloning the repository and having Docker installed on your host machine*

```
docker build . -t tfci:tagname
```

## Running the TFCI Container (Environment Variables, WORKDIR, Bind mount)

### Environment Variables `-e` or `-env`

Tfci requires that the following values be passed in as environment variables from the host machine **or** injected using the global command --flags
- `TF_HOSTNAME` (for TFE)
- `TF_API_TOKEN`
- `TF_CLOUD_ORGANIZATION`

#### Available Environment Variables

| ENV Var Name      | Default            | Flag            |  Description                                                                                                     |
| ----------------- |--------------------|-----------------| ---------------------------------------------------------------------------------------------------------------- |
| `TF_HOSTNAME`     | `app.terraform.io` |  `--hostname`     | The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to Terraform Cloud. |
| `TF_API_TOKEN`    | `n/a`              |  `--token`        | The token used to authenticate with Terraform Cloud. [API Token Docs](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens)                                                           |
| `TF_ORGANIZATION` | `n/a`              |  `--organization` | The name of the organization in Terraform Cloud.                                                                 |
| `TF_MAX_TIMEOUT`  | `1h`               |  N/A            | Max wait timeout to wait for actions to reach desired or errored state. ex: `1h30`, `30m`                                         |
| `TF_VAR_*`        | `n/a`              |  N/A            | Only applicable for create-run action. Note: strings must be escaped. ex: `TF_VAR_image_id="\"ami-abc123\""`. All values must be expressed as an HCL literal in the same syntax you would use when writing Terraform code. [Create Run API Docs](https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#create-a-run)                                 |
| `TF_LOG`          | `OFF`              |  N/A            | Debugging log level options: `OFF`, `ERROR`, `INFO`, `DEBUG`                                                     |


**Docker environment variable example**
```sh
docker run -it --rm \
-e "TF_HOSTNAME" \
-e "TF_API_TOKEN" \
-e "TF_CLOUD_ORGANIZATION" \
hashicorp/tfci:latest \
tfci run show --help
```

**Or pass these values as global flags to tfci**

```sh
docker run -it --rm \
hashicorp/tfci:latest \
tfci \
--hostname="..." \
--organization="..." \
--token="..." \
run show --help
```

### Workdir and Bind mount

Since tfci is executing within a Docker container, the `upload` command needs to access your repository's configuration directory declared with the `--directory` flag on the host machine.

In order to configure this, you will need to set both a Working directory for the container as well as a bind mount from the host machine.

The `WORKDIR` or `--workdir` in the container can be any path, even one that does not exist within container. Example `--workdir "/tfci/workspace"`, docker will create the filesystem path in the container for you.

Use the `--volumes` or `-v` flag, to create a bind mount between the container working directory and filesystem path on the host machine containing the terraform configuration to upload

#### Example

Say your project lives withing the following path on the host, `/path/to/your/configuration`.
```
├── terraform/
|   └── main.tf
└── README.md
```

When running the tfci container, create a bind mount between your Terraform configuration project and the container working directory like the following.

```bash
docker run -it --rm \
-e "TF_HOSTNAME" \
-e "TF_API_TOKEN" \
-e "TF_CLOUD_ORGANIZATION" \
-e "TF_LOG" \
--workdir "/tfci/workspace" \
-v "/path/to/your/configuration":"/tfci/workspace" \
hashicorp/tfci:latest \
tfci upload --workspace=api-workspace --directory=./terraform
```
Since the bind mount is between the host project root directory and container working directory, you can pass the the relative path to the configuration you wish to upload to Terraform Cloud.

### Piping Json Output

While executing tfci within a Docker container, avoid the Docker `-it` flag, which allocates a pseudo-TTY connected to the container's stdin.

This can break when piping the stdout from tfci to other programs such as `jq`.

## Troubleshooting

Recommend to set the environment variable: `TF_LOG` to `DEBUG` level to inspect additional diagnostics or error information.

## Local Development

Recommend to use a environment shell tool such as [direnv](https://direnv.net/)

#### GitHub Action Runner environment variable mocking

*If using .envrc, can add the following exported values to your shell to test behavior when running in GitHub Actions.*

These environment variables are passed to custom Docker GitHub Actions

*.envrc*
```sh
export TF_HOSTNAME="my-enterprise-tf-instance.example.com"
export TF_CLOUD_ORGANIZATION="<redacted>"
export TF_API_TOKEN="<redacted>"
# GitHub ENV
export CI=true
export GITHUB_ACTIONS="true"
export GITHUB_OUTPUT=./tmp
export GITHUB_SHA=13c988d4f15e06bcdd0b0af290086a3079cdadb0
export GITHUB_ACTOR=octocat
export GITHUB_RUN_ID=8675309
export GITHUB_RUN_NUMBER=5
```

Then run the container passing the environment variables as so

```sh
docker run -it \
  -e "TF_HOSTNAME" \
  -e "TF_API_TOKEN" \
  -e "TF_CLOUD_ORGANIZATION" \
  -e "CI" \
  -e "GITHUB_ACTIONS" \
  -e "GITHUB_OUTPUT" \
  -e "GITHUB_SHA" \
  -e "GITHUB_ACTOR" \
  -e "GITHUB_ACTOR" \
  hashicorp/tfci:latest \
  tfci run show --help
```

## Usage with Terraform Enterprise
If Terraform Enterprise is using TLS certificates signed by a private CA build a custom image.

1. Create a directroy named `docker-tfci-custom`
2. Change into that directory
3. Create a file in that directory named `Dockerfile`
4. Add the following and substitute `/path/to/ca.crt` for the path to your CA certificate:

```dockerfile
FROM hashicorp/tfci:latest
COPY /path/to/ca.crt /usr/local/share/ca-certificates/
RUN apk --no-cache add ca-certificates
RUN update-ca-certificates
```

5. Build the image from the docker file like so replacing `registry.example.com/namespace` with your container registries address and namespace and whatever you wish to name the container.

```sh
docker build -t registry.example.com/namespace/tfci-custom .
```

> Note: If you are using GitLab as your container registry see [Naming Convention for Container Images](https://docs.gitlab.com/ee/user/packages/container_registry/#naming-convention-for-your-container-images)

Push the custom container to your container registry like so

```sh
docker push registry.example.com/namespace/tfci-custom
```

## Generating a binary from source

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
