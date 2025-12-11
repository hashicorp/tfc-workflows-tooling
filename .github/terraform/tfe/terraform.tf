# Copyright IBM Corp. 2023, 2025
# SPDX-License-Identifier: MPL-2.0

terraform {
  cloud {} // omitted for env variables
  required_providers {
    tfe = {
      source = "hashicorp/tfe"
    }
  }
}
