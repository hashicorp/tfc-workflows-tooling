## Release Process

Changes such as documentation updates or test fixes, do NOT require a release. You can merge changes into main once they have passed CI and approved. As soon as new fixes and features are merged, a new release can be made. Once a new release has been made and is successful, a release will need to be created for [hashicorp/tfc-workflows-github](https://github.com/hashicorp/tfc-workflows-github) and [hashicorp/tfc-workflows-gitlab](https://github.com/hashicorp/tfc-workflows-gitlab) respectively.

### Preparing a release

Find the latest release tag and compare with the main branch to fully understand the changes to be released. You can compare the last release tag with main with the following [example](`https://github.com/hashicorp/tfc-workflows-tooling/compare/v1.1.1...main`).

For each meaningful change, check the following:

1. Each change, except for documentation and tests, is added to CHANGELOG.md.
1. If there are new features and enhancements, consider vetting changes locally or use the repository’s [GitHub test actions](https://github.com/hashicorp/tfc-workflows-tooling/tree/main/.github/actions) to manually test the new functionality.
  * For new commands, will need to create a new action.yaml.
  * For existing commands, if a new option is added, will need to add that to the existing test action.yml file.

Prepare the changelog for a new release:

1. Replace `# Unreleased` with the version you are missing.
1. Ensure to add a new `# Unreleased section` at the top of the changelog for future changes. This will make it clearer to authors where to add their changelog entry.
1. Ensure each existing changelog entry for the new release has the author(s) attributed and a pull request linked. i.e `- Some new feature/bugfix by @some-github-user (#3)[link-to-pull-request]`
1. Open a pull request with these changes with a title similar to `vX.XX.XX Changelog`. Once approved and merged, you can go ahead and create the release.

### Creating a release

1. [Create a new release in GitHub](https://help.github.com/en/github/administering-a-repository/creating-releases) by clicking “Releases“ and “Draft a new release”
2. Set the `Tag Version` to a new tag, using [Semantic Versioning](https://semver.org/)
3. Set the `Target` as `main`.
4. Set the `Release Title` to the tag you created, `vX.X.X`
5. Use the description section to describe the changes. Make sure to include the changelog entries for this release with the author(s) attributed and pull request linked.

Use the following headers in the description of your release:
* `Breaking Changes`: Use this for any changes that aren't backwards compatible. Include details on how tot handle these changes.
* `Features`: Use this for any large new features that are added.
* `Enhancements`: Use this for smaller new features that are added.
* `Bug Fixes`: Use this for nay bugs that were fixed.
* `Notes`: Use this for additional notes regarding upcoming deprecations, or other information to highlight.

Markdown Examples:

```
## Enhancements
* Add description of new small feature by @some-github-user (#7)[link-to-pull-request]

## Bug Fixes
* Fix description for a bug by @some-github-user (#7)[link-to-pull-request]
```

6. Click “Publish release” to save and publish your release.
7. Monitor the [Build and Release GitHub Workflow](/.github/workflows/build-release.workflow.yml) and verify it has succeeded before proceeding with any downstream dependent releases.


### GitHub Actions Release Process: [hashicorp/tfc-workflows-github](https://github.com/hashicorp/tfc-workflows-github)

#### Preparing a release

Find the latest release tag and compare with the main branch to fully understand the changes to be released. You can compare the last release tag with main with the following example for [tfc-workflows-github](https://github.com/hashicorp/tfc-workflows-github/compare/v1.1.1...main).

For each meaningful change, check the following:
1. Each change, except for documentation, is added to CHANGELOG.md.
2. If there are new features and enhancements in the form of either new commands or new command options, the relevant */action.yml files are updated to reflect the changes.
  * Before preparing a release, create a separate PR including the changes for new actions or updates to existing actions if the main branch . Use a separate project to test the requested changes using your branch as the action version. i.e upload-configuration@some-user/test-branch.

Prepare the changelog for a new release:
1. Replace `# Unreleased` with the version you are missing.
2. Ensure to add a new `# Unreleased` section at the top of the changelog for future changes. This will make it clearer to authors where to add their changelog entry.
3. Ensure each existing changelog entry for the new release has the author(s) attributed and a pull request linked. i.e `- Some new feature/bugfix by @some-github-user (#3)[link-to-pull-request]`
4. Open a pull request with these changes with a title similar to `vX.XX.XX Changelog`. Once approved and merged, you can go ahead and create the release.

#### Creating a release

1. [Create a new release in GitHub](https://help.github.com/en/github/administering-a-repository/creating-releases) by clicking “Releases“ and “Draft a new release”
2. Set the Tag Version to a new tag, preferably using the same tag version as [TFCI]( ) to keep the projects consistent.  [More on Semantic Versioning](https://semver.org/).
3. Set the `Target` as `main`.
4. Set the `Release Title` to the tag you created, `vX.X.X`
5. Use the description section to describe the changes. Make sure to include the changelog entries for this release with the author(s) attributed and pull request linked.

Use the following headers in the description of your release:
* `Breaking Changes`: Use this for any changes that aren't backwards compatible. Include details on how tot handle these changes.
* `Features`: Use this for any large new features that are added.
* `Enhancements`: Use this for smaller new features that are added.
* `Bug Fixes`: Use this for nay bugs that were fixed.
* `Notes`: Use this for additional notes regarding upcoming deprecations, or other information to highlight.

Markdown Examples:

```
## Enhancements
* Add description of new small feature by @some-github-user (#7)[link-to-pull-request]

## Bug Fixes
* Fix description for a bug by @some-github-user (#7)[link-to-pull-request]
```

6. Click “Publish release” to save and publish your release.


### GitLab Pipelines Release Process: [hashicorp/tfc-workflows-gitlab](https://github.com/hashicorp/tfc-workflows-gitlab)

#### Preparing a release

Find the latest release tag and compare with the main branch to fully understand the changes to be released. You can compare the last release tag with main with the following example for [tfc-workflows-gitlab](`https://github.com/hashicorp/tfc-workflows-gitlab/compare/v1.1.1...main `).

For each meaningful change, check the following:
1. Each change, except for documentation, is added to CHANGELOG.md.
2. If there are new features and enhancements in the form of either new commands or new command options, the relevant yml workflow files are updated to reflect the changes.
  * Before preparing a release, create a separate PR including the changes for new actions or updates to existing actions if the main branch . Use a separate project to test the requested changes using your branch as the action version. i.e upload-configuration@some-user/test-branch.

Prepare the changelog for a new release:

1. Replace `# Unreleased `with the version you are missing.
2. Ensure to add a new `# Unreleased` section at the top of the changelog for future changes. This will make it clearer to authors where to add their changelog entry.
3. Ensure each existing changelog entry for the new release has the author(s) attributed and a pull request linked. i.e `- Some new feature/bugfix by @some-github-user (#3)[link-to-pull-request]`
4. Open a pull request with these changes with a title similar to `vX.XX.XX Changelog`. Once approved and merged, you can go ahead and create the release.

#### Creating a release

1. [Create a new release in GitHub](https://help.github.com/en/github/administering-a-repository/creating-releases) by clicking “Releases“ and “Draft a new release”
2. Set the `Tag Version` to a new tag, preferably using the same tag version as [TFCI]( ) to keep the projects consistent.  [More on Semantic Versioning](https://semver.org/).
3. Set the `Target` as `main`.
4. Set the `Release Title` to the tag you created, `vX.X.X`
5. Use the description section to describe the changes. Make sure to include the changelog entries for this release with the author(s) attributed and pull request linked.

Use the following headers in the description of your release:
* `Breaking Changes`: Use this for any changes that aren't backwards compatible. Include details on how tot handle these changes.
* `Features`: Use this for any large new features that are added.
* `Enhancements`: Use this for smaller new features that are added.
* `Bug Fixes`: Use this for nay bugs that were fixed.
* `Notes`: Use this for additional notes regarding upcoming deprecations, or other information to highlight.

Markdown Examples:

```
## Enhancements
* Add description of new small feature by @some-github-user (#7)[link-to-pull-request]

## Bug Fixes
* Fix description for a bug by @some-github-user (#7)[link-to-pull-request]
```

6. Click “Publish release” to save and publish your release.
