pub async fn get_vendor(device: udev::Device) -> super::GPUVendor {
	match device.attribute_value("vendor") {
		Some(v)	=> {
			return super::map_to_vendor(v, &device)
		}
		None	=> {}
	};

	let parent = match device.parent() {
		Some(v)	=> {v}
		None	=> {return super::GPUVendor::Others}
	};

	match parent.attribute_value("vendor") {
		Some(v)	=> {
			return super::map_to_vendor(v, &device);
		}
		None	=> {
			return super::GPUVendor::Others;
		}
	}
}
