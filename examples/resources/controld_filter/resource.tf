resource "controld_filter" "malware" {
  profile_id = controld_profile.home.id
  filter     = "malware"
  status     = true
}
