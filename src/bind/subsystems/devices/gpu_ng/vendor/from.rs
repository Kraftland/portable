/**
	This function is used by trait implementation From<OsStr> for GPUVendor.

	It takes a borrowed parent udev Device to gather information about the graphics card.
*/
fn device_to_vendor(device: &udev::Device) -> super::GPUVendor {
	let vendor = match device.property_value("vendor") {
		Some(v)	=> v,
		None	=> return super::GPUVendor::Other { vendor: format!("None") }
	};

	match vendor.to_str() {
		Some("0x8086")	=> super::GPUVendor::Intel,
		Some("0x1002")	=> super::GPUVendor::AMD,
		Some("0x10de")	=> {
			super::GPUVendor::NVIDIA { driver: device.driver().into() }
		}

		Some(_) | None	=> super::GPUVendor::Other {
			vendor:	vendor.to_string_lossy().to_string(),
		}
	}
}
