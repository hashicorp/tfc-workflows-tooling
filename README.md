# HCP Terraform Workflows Tooling

HCP Terraform Workflows Tooling is a dockerized go application to automate HCP Terraform Runs via the API.

## Supported Platforms

* GitHub Actions
* GitLab Pipelines

## Features

* **Run Management**: Create, apply, show, discard, and cancel Terraform runs
* **Plan Operations**: Output plan results for review
* **Workspace Outputs**: List and retrieve workspace output values
* **Policy Operations**: Evaluate Sentinel policies and apply overrides with justification
* **Configuration Upload**: Upload Terraform configurations to HCP Terraform

## Usage

See [`docs/USAGE.md`](https://github.com/hashicorp/tfc-workflows-tooling/blob/main/docs/USAGE.md)

### Quick Example - Policy Operations

```bash
# Check policy evaluation results
tfci policy show --run-id run-abc123

# Override mandatory policy failures with justification
tfci policy override \
  --run-id run-abc123 \
  --justification "Emergency hotfix approved by CTO - INC-12345"
```

## Related Projects

* [tfc-workflows-github](https://github.com/hashicorp/tfc-workflows-github)
* [tfc-workflows-gitlab](https://github.com/hashicorp/tfc-workflows-gitlab)

## Contributing Guideline

See [`docs/CONTRIBUTING.md`](https://github.com/hashicorp/tfc-workflows-tooling/blob/main/docs/CONTRIBUTING.md)

## License

[Mozilla Public License v2.0](https://github.com/hashicorp/tfc-workflows-tooling/blob/main/LICENSE)
