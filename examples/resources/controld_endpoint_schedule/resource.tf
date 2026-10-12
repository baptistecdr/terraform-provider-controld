resource "controld_endpoint_schedule" "bedtime" {
  name      = "Kids bedtime"
  device_id = controld_device.kids_tablet.id
  time_zone = "Europe/Paris"

  # Enforce the strict profile on weekday evenings.
  windows = [
    for day in ["mon", "tue", "wed", "thu", "fri"] : {
      weekday    = day
      time_start = "21:00"
      time_end   = "23:59"
      profile_id = controld_profile.strict.id
    }
  ]
}
