resource "controld_profile" "home" {
  name = "Home Network"
}

resource "controld_profile" "guest" {
  name             = "Guest Network"
  clone_profile_id = controld_profile.home.id
}
