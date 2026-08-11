pub fn get_paths() -> Vec<std::path::PathBuf> {
	vec![
		"/proc/bus".into(),
		"/proc/driver".into(),

		"/sys/devices/virtual/dmi".into(),
		"/sys/devices/virtual/block".into(),
		"/sys/devices/virtual/sound".into(),

		"/etc/kernel".into(),

		"/proc/1".into(),
		"/usr/share/applications".into(),
	]
}
