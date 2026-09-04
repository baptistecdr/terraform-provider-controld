resource "controld_custom_rule" "block_ads" {
  profile_id = controld_profile.home.id
  hostname   = "ads.example.com"
  do         = 0 # block
  status     = true
  group      = controld_rule_folder.ads.id
  comment    = "Known ad server"
}
