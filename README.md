<h3 align="center">terraform-provider-controld</h3>
<p align="center">
    A Terraform provider for managing <a href="https://controld.com/">ControlD</a> DNS profiles, devices,
    custom rules, filters, and services, built on top of
    <a href="https://github.com/baptistecdr/controld-go">controld-go</a>.
    <br>
    <a href="https://github.com/baptistecdr/terraform-provider-controld/issues/new">Report bug</a>
    ·
    <a href="https://github.com/baptistecdr/terraform-provider-controld/issues/new">Request feature</a>
</p>

<div align="center">

[![Tests](https://github.com/baptistecdr/terraform-provider-controld/actions/workflows/test.yml/badge.svg)](https://github.com/baptistecdr/terraform-provider-controld/actions/workflows/test.yml)
![GitHub Tag](https://img.shields.io/github/v/tag/baptistecdr/terraform-provider-controld?label=Latest%20version)

</div>

This is an unofficial provider for [ControlD](https://controld.com/), built on the
[Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). It isn't affiliated
with ControlD.

## Features

- `controld_profile` — manage DNS profiles
- `controld_device` — manage devices and the profile(s) applied to them
- `controld_default_rule` — manage a profile's default (fallback) action
- `controld_rule_folder` — manage custom rule folders (groups)
- `controld_custom_rule` — manage hostname-based custom rules (block, bypass, spoof, redirect)
- `controld_service` — toggle built-in services (streaming, social media, ...) per profile
- `controld_filter` — enable/disable native filter lists (malware, ads, ...) per profile
- Data sources for profiles, devices, rule folders, services, native/external filters, and account
  (`controld_user`) information

## Quick start

```terraform
terraform {
  required_providers {
    controld = {
      source = "baptistecdr/controld"
    }
  }
}

provider "controld" {
  # api_token can also be set via the CONTROLD_API_TOKEN environment variable.
  api_token = var.controld_api_token
}

resource "controld_profile" "home" {
  name = "Home Network"
}

resource "controld_device" "laptop" {
  name       = "My Laptop"
  profile_id = controld_profile.home.id
  icon       = "desktop-mac"
}
```

See the [`docs/`](docs/) directory for the full list of resources and data sources.

## How to build

- Install [Go](https://golang.org/doc/install) >= 1.21
- Clone the project
- Run `go install`

## Development

- Run `go generate` to format the example configs and regenerate `docs/`
- Run `make testacc` to run the acceptance test suite (creates real resources against the ControlD
  account tied to `CONTROLD_API_TOKEN`)

## Bugs and feature requests

Have a bug or a feature request? Please first search for existing and closed issues. If your problem or
idea is not addressed yet, [please open a new issue](https://github.com/baptistecdr/terraform-provider-controld/issues/new).

## Contributing

Contributions are welcome!

## Thanks to

- https://github.com/baptistecdr/controld-go — the Go client this provider is built on
- https://github.com/hashicorp/terraform-provider-scaffolding-framework — the project scaffold this
  provider was built on
- Claude
