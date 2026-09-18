
/// True if expose_list is in the scope of rw, ro and device
pub fn permission_contained(
	rw:		&Vec<std::path::PathBuf>,
	ro:		&Vec<std::path::PathBuf>,
	device:		&Vec<std::path::PathBuf>,
	expose_list:	&Vec<crate::pref::runtime::options::FileExposurePreference>,
) -> bool {
	for expose in expose_list {
		match expose {
			crate::pref::runtime::options::FileExposurePreference::MountPath { host, dest: _, class }
			=> {
				match class {
					crate::bind::types::BindType::Device	=> {
						if ! device.contains(host) {
							return false;
						}
					}
					crate::bind::types::BindType::ReadOnly	=> {
						if ! ro.contains(host) {
							return false;
						}
					}
					crate::bind::types::BindType::ReadWrite	=> {
						if ! rw.contains(host) {
							return false;
						}
					}
				}
			}
			crate::pref::runtime::options::FileExposurePreference::Passthrough { host }
			=> {
				if rw.contains(host) {
					continue;
				} else {
					return false;
				}
			}
		}
	};

	true
}
