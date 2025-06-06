# UNRELEASED

# v1.4.0

## Enhancements
* Adds support for `-refresh` option to the `run create` command by @mpucholblasco [#154](https://github.com/hashicorp/tfc-workflows-tooling/pull/154)

# v1.3.3
* Bumps tfci go version to `v1.23` by @mjyocca [#141](https://github.com/hashicorp/tfc-workflows-tooling/pull/141)
* Go Dependency cleanup (`tidy`) by @mjyocca [#143](https://github.com/hashicorp/tfc-workflows-tooling/pull/143)

# v1.3.2

## Bug Fixes
* Fix issue with `run create` command not handling post-plan status by @sam-mosleh [#137](https://github.com/hashicorp/tfc-workflows-tooling/pull/137)

# v1.3.1

## Notes
* Updates documentation and references from "Terraform Cloud" to "HCP Terraform" by @mjyocca [#107](https://github.com/hashicorp/tfc-workflows-tooling/pull/107)
* Fix links to starter template files in ADOPTION.md file by @Rohlik [#118](https://github.com/hashicorp/tfc-workflows-tooling/pull/118)

## Bug Fixes
* Compiles for Linux regardless of current CPU architecture when using the provided Dockerfile by @ggambetti [#113](https://github.com/hashicorp/tfc-workflows-tooling/pull/113)

# v1.3.0

## Enhancements
* Adds support for Terraform target under `run create` command by @trutled3 [#97](https://github.com/hashicorp/tfc-workflows-tooling/pull/97)

## Bug Fixes
* Fixes an issue with recently added target argument for `run create` command by @mjyocca [#101](https://github.com/hashicorp/tfc-workflows-tooling/pull/101)

# v1.2.0

## Enhancements
* Add support for saved plans by @1newsr [#57](https://github.com/hashicorp/tfc-workflows-tooling/pull/57)
* Adds support for Terraform destroy under `run create` command by @trutled3 [#80](https://github.com/hashicorp/tfc-workflows-tooling/pull/80)

## Additional changes
* Adds the tfci binary to .gitignore [#81](https://github.com/hashicorp/tfc-workflows-tooling/pull/81)

# v1.1.1

## Enhancements
* Adds support for a `--json` flag option for across all commands by @mjyocca [#58](https://github.com/hashicorp/tfc-workflows-tooling/pull/58)

## Bug Fixes
* Fixes issue with `workspace output list` not including output json data to platform specific output by @mjyocca [#60](https://github.com/hashicorp/tfc-workflows-tooling/pull/60)

# v1.1.0

## Enhancements
* Adds new command, `workspace output list` to retrieve outputs for an existing Terraform Cloud workspace by @mjyocca [#29](https://github.com/hashicorp/tfc-workflows-tooling/pull/29)

## Bug Fixes
* Fixes issue with `payload` output missing from `run create`, `run show`, `upload`, `plan output` commands by @mjyocca [#46](https://github.com/hashicorp/tfc-workflows-tooling/pull/46)
* Fixes race condition with the `run create` command when an `auto-apply` configured TFC/TFE workspace exits too soon by @mjyocca [#55](https://github.com/hashicorp/tfc-workflows-tooling/pull/55)

# v1.0.4 (patch-v1.0.4)

## Bug Fixes
* Fixes issue with `payload` output missing from `run create`, `run show`, `upload`, `plan output` commands by @mjyocca [#46](https://github.com/hashicorp/tfc-workflows-tooling/pull/46)

# v1.0.3

## Enhancements
* Upgrades [go-tfe](https://github.com/hashicorp/go-tfe) to `v1.30.0`.
* Internal refactor for output interface by @mjyocca [#22](https://github.com/hashicorp/tfc-workflows-tooling/pull/22)
* Adds additional error messages when encountering issues with TFC/E requests by @mjyocca [#28](https://github.com/hashicorp/tfc-workflows-tooling/pull/28)

## Bug Fixes
* Fixes issue with runs incorrectly marked as `Error` when status ends in `policy_soft_failed` by @mjyocca [#23](https://github.com/hashicorp/tfc-workflows-tooling/pull/23)
* Fixes bug with reading logs from unreached sentinel policies blocking the main go-routine by @mjyocca [#24](https://github.com/hashicorp/tfc-workflows-tooling/pull/24)

# v1.0.2

## Bug Fixes

* Upgrades [go-tfe](https://github.com/hashicorp/go-tfe) to `v1.28.0` to [avoid sending credentials during ConfigurationVersion upload](https://github.com/hashicorp/go-tfe/pull/717), as they are not necessary.

# v1.0.1

## Enhancements
* Adds [mitchellh/cli](https://github.com/mitchellh/cli) `Command.Synopsis()` to all commands by @ggambetti [#14](https://github.com/hashicorp/tfc-workflows-tooling/pull/14)

## Bug Fixes
* Fixes `run create` command for Auto Apply workspaces by @mjyocca [#16](https://github.com/hashicorp/tfc-workflows-tooling/pull/16)

# v1.0.0

First Release
