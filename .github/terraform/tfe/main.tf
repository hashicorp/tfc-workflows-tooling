# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "tfe_organization" "org" {
  name = var.organization
}

resource "tfe_workspace" "ci-workspace-test" {
  name         = "ci-workspace-test"
  organization = data.tfe_organization.org.name
  tag_names    = ["test", "ci", "a"]
}
