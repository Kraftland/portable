pub async fn get_vendor(device: udev::Device) -> GPUVendor {
	match device.attribute_value("vendor") {
		Some(v)	=> {
			return map_to_vendor(v, &device)
		}
		None	=> {}
	};

	let parent = match device.parent() {
		Some(v)	=> {v}
		None	=> {return GPUVendor::Others}
	};

	match parent.attribute_value("vendor") {
		Some(v)	=> {
			return map_to_vendor(v, &device);
		}
		None	=> {
			return GPUVendor::Others;
		}
	}
}
