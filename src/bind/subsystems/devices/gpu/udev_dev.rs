pub async fn get_vendor(device: udev::Device) -> super::GPUVendor {
	match device.attribute_value("vendor") {
		Some(v)	=> {
			return map_to_vendor(v, &device)
		}
		None	=> {}
	};

	let parent = match device.parent() {
		Some(v)	=> {v}
		None	=> {return super::GPUVendor::Others}
	};

	match parent.attribute_value("vendor") {
		Some(v)	=> {
			return map_to_vendor(v, &device);
		}
		None	=> {
			return super::GPUVendor::Others;
		}
	}
}

fn map_to_vendor(vendor_string: &std::ffi::OsStr, device: &udev::Device) -> super::GPUVendor {
	use super::GPUVendor;
	let string = vendor_string.to_str().unwrap_or("unknown");
	match string {
		"0x8086"	=> {GPUVendor::Intel}
		"0x10de"	=> {
			GPUVendor::NVIDIA {
				driver: super::nvidia::NVIDIADriver::get(device),
			}
		}
		"0x1002"	=> {GPUVendor::AMD}
		_		=> {GPUVendor::Others}
	}
}
