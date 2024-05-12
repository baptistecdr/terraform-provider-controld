resource "controld_device" "laptop" {
  name       = "My Laptop"
  profile_id = controld_profile.home.id
  icon       = "desktop-mac"
  learn_ip   = true
}
