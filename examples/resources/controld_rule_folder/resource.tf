resource "controld_rule_folder" "ads" {
  profile_id = controld_profile.home.id
  name       = "Ads & Trackers"
  do         = 0 # block
  status     = true
}
