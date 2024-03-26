# Adopting TFCI

If you have any questions or need clarification feel free to open an [issue](https://github.com/hashicorp/tfc-workflows-tooling/issues).

## Setup

Tfci currently supports the following CI/CD platforms:
* [GitHub Actions](https://docs.github.com/en/actions)
* [GitLab Pipelines](https://docs.gitlab.com/ee/ci/pipelines/)

Tfci can be instrumented for other platforms with the use of the [published Docker Container](https://hub.docker.com/r/hashicorp/tfci).

### Docker

Tfci generates an Docker container artifact that is available from the Docker public registry, `docker://hashicorp/tfci:{VERSION}`.

Leveraging the publicly distributed Docker image is the recommended approach for running Tfci on CI/CD platforms for non-human interactions.

View our [Usage documentation](./USAGE.md) to learn more our available commands, arguments, and configuration.

### Building a binary from source code

If Docker is not available, view our [Usage documentation](./USAGE.md#generating-a-binary-from-source) to learn more.

### How GitHub Actions uses Tfci

[View all](https://github.com/hashicorp/tfc-workflows-github/tree/main/actions) available GitHub Actions that are built on top of Tfci.

### [How GitLab Pipelines uses Tfci](https://github.com/hashicorp/tfc-workflows-gitlab)

View the GitLab [Base-Template](https://github.com/hashicorp/tfc-workflows-gitlab/blob/main/Base.gitlab-ci.yml)

## Workflow

### [Terraform Cloud CLI](https://developer.hashicorp.com/terraform/cloud-docs/run/cli) vs. [Terraform Cloud API](https://developer.hashicorp.com/terraform/cloud-docs/run/api)

| Workflow   |    Terraform CLI (Cloud)            |  TFCI/Terraform Cloud API                      |
|------------|-------------------------------------|------------------------------------------------|
| Plan       |  `terraform plan`                   |  commands: `upload`, `run create`              |
| Apply      |  `terraform apply -auto-approve`    |  commands: `upload`,  `run create`, `run apply`|
| Destroy    |  `terraform plan -destroy -out=destroy.tfplan` , `terraform apply destroy.tfplan`| commands: `run create -is-destroy=true` |
| Target     | `terraform plan -target aws_instance.foo` | commands: `run create -target=aws_instance.foo` |

#### Terraform Plan

Terraform Cloud CLI can execute a new plan with one command that will upload Terraform configuration and execute a new run in Terraform Cloud.

With Tfci and Terraform Cloud API driven runs, these actions are broken up into multiple parts:
- Upload terraform configuration as a ConfigurationVersion
- Create a new run using that Configuration Version. If the run was not specified as `plan-only`, then it could be optionally approved or applied.
- Focus Terraform's attention on only a subset of resources with the `target` option.

#### Terraform Apply

Terraform Cloud CLI can execute an apply run with, `terraform apply` that will also upload the configuration and start a new Terraform Cloud run that will plan and apply.

With Tfci and Terraform Cloud API driven runs:
- Upload terraform configuration as a ConfigurationVersion
- New plan run executes
- If plan phase was successful, an apply can be confirmed to proceed

#### Terraform Destroy

Terraform Cloud CLI can execute a destroy run with, `terraform plan -destroy -out=destroy.tfplan` followed by `terraform apply destroy.tfplan`, that will also upload the configuration and start a new Terraform Cloud run that will plan and destroy.

With Tfci and Terraform Cloud API driven runs:
- Upload terraform configuration as a ConfigurationVersion
- New plan run executes
- If plan phase was successful, an apply can be confirmed to proceed

### Prescribed Workflows

As part of the initiative for this project and Tfci, we prescribe the following recommended workflows:
* Speculative Run
* Plan/Apply Run

#### Speculative Run

*This is a plan-only run. It is generally used on a Pull/Merge Request.*

[Example speculative run created for GitHub Actions](https://github.com/hashicorp/tfc-workflows-github/blob/main/workflow-templates/terraform-cloud.speculative-run.workflow.yml)

Steps:
1. Pull request/Merge request is opened and triggers the workflow.
1. Pipeline runner checkouts the branch with the included changes
1. Upload Configuration, optionally mark it as `speculative` so the same configuration version cannot be applied.
1. A new run starts in Terraform Cloud for the uploaded ConfigurationVersion, specifying the run as `plan-only`. This creates a speculative run that does not lock the workspace.
1. Results of the run and plan is presented to the user, such as a comment on PR/Merge request.


#### Plan/Apply Run

*This is a run that is both planned and applied. It is generally recommended on merges.*

[Example apply-run created for GitHub Actions](https://github.com/hashicorp/tfc-workflows-github/blob/main/workflow-templates/terraform-cloud.apply-run.workflow.yml)

Steps:
1. Code is merged into the default main branch or branch used for a separate environment such as "staging".
1. Pipeline runner checks out the merged code.
1. Configuration is uploaded to Terraform Cloud.
1. A new run starts in Terraform Cloud for the uploaded ConfigurationVersion.
1. If the plan was successful, confirm the run by running `run apply --run={previous run id}`, that will then apply the changes
1. Optionally output the apply results as a summary.
