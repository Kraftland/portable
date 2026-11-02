/**
	This public function allows extracting PathBufs from a borrowed udev device.

	Note that this breaks with non-UTF-8 paths, because we internally converts them into Strings
*/
pub fn get(device: &udev::Device) -> Result<Vec<std::path::PathBuf>, crate::bind::subsystems::devices::gpu_ng::enumerate::EnumerateError> {
	let raw_devlinks = match device.property_value("DEVLINKS") {
		Some(v)	=> v,
		None	=> {
			return Ok(vec![]);
		}
	};

	let raw_devlinks = match raw_devlinks.to_str() {
		Some(v)	=> v,
		None	=> {
			use crate::bind::subsystems::devices::gpu_ng::enumerate::EnumerateError;
			return Err(
				EnumerateError::DevlinksNonUnicode(raw_devlinks.to_owned())
			);
		}
	};

	let splitted_entries = raw_devlinks.split(" ");

	let mut ret = vec![];

	for entry in splitted_entries {
		ret.push(std::path::PathBuf::from(entry));
	};

	Ok(ret)
}
