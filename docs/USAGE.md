# Usage

## Available Commands

* `upload`: Creates and uploads configuration files for a given workspace
* `run show`: Returns run details for the provided Terraform Cloud Run ID.
* `run create`: Performs a new plan run in Terraform Cloud, using a configuration version and the workspace's current variables.
* `run apply`: Applies a run that is paused waiting for confirmation after a plan.
* `run discard`: Skips any remaining work on runs that are paused waiting for confirmation or priority.
* `run cancel`: Interrupts a run that is currently planning or applying.
* `plan output`: Returns the plan details for the provided Plan ID.

## Pulling Image from Dockerhub

Pulling the latest version
```sh
docker pull hashicorp/tfci:latest
```

Pulling a specific version

```sh
docker pull hashicorp/tfci:v1.0.0
```

Then run the container.

You are able to pass exported environment variables `TF_HOSTNAME` (if using Terraform Enterprise), `TF_API_TOKEN`, `TF_CLOUD_ORGANIZATION` from the parent process.

```sh
docker run -it --rm \
  -e "TF_HOSTNAME" \
  -e "TF_API_TOKEN" \
  -e "TF_CLOUD_ORGANIZATION" \
  hashicorp/tfci:latest \
  tfci run show --help
```

Or pass these variables as global flags to the tfci command.

```sh
docker run -it --rm \
  hashicorp/tfci:latest \
  tfci \
  --hostname="..." \
  --organization="..." \
  --token="..." \
  run show --help
```

## Building Image Locally

```
docker build . -t tfci:tagname
```

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
