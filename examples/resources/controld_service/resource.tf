resource "controld_service" "netflix" {
  profile_id = controld_profile.home.id
  service    = "netflix"
  do         = 1 # bypass
  status     = true
}
