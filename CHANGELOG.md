# UNRELEASED

## Enhancements

## Bug Fixes
* Fixes issue with runs incorrectly marked as `Error` when status ends in `policy_soft_failed` by @mjyocca [#23](https://github.com/hashicorp/tfc-workflows-tooling/pull/23)

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
