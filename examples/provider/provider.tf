terraform {
  required_providers {
    controld = {
      source = "baptistecdr/controld"
    }
  }
}

# Configuration-based authentication.
provider "controld" {
  api_token = var.controld_api_token
}
