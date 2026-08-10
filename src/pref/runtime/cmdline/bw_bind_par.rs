/**
	Get an exposure table from legacy environment variable bwBindPar

	This variable is single path only.
*/
pub fn get_bwbindpar_opts() -> Option<crate::pref::runtime::options::FileExposurePreference> {
	use crate::pref::runtime::options::FileExposurePreference;
	use std::path::PathBuf;
	match std::env::var("bwBindPar") {
		Ok(v)	=> {
			Some(
				FileExposurePreference::MountPath {
					host: PathBuf::from(&v),
					dest: PathBuf::from(v),
					class: crate::bind::types::BindType::ReadWrite,
				}
			)
		}
		Err(_)	=> {None}
	}
}
