resource "controld_default_rule" "home" {
  profile_id = controld_profile.home.id
  do         = 1 # bypass
  status     = true
}
