
/// True if stored variants are in the scope of rw, ro and device
pub fn permission_contained(
	rw:		&Vec<super::MountPathPreference>,
	ro:		&Vec<super::MountPathPreference>,
	device:		&Vec<super::MountPathPreference>,

	rw_stored:	&Vec<std::path::PathBuf>,
	ro_stored:	&Vec<std::path::PathBuf>,
	dev_stored:	&Vec<std::path::PathBuf>,
) -> bool {
	for path in rw {
		if rw_stored.contains(&path.0) {
			continue;
		} else {
			return false;
		}
	};

	for path in ro {
		if ro_stored.contains(&path.0) {
			continue;
		} else {
			return false;
		}
	};

	for path in device {
		if dev_stored.contains(&path.0) {
			continue;
		} else {
			return false;
		}
	};

	true
}
