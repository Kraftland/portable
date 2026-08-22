
/**
	Gather GPU information and return a complete struct Info.
*/
pub async fn get(dev: super::GPUDevice) -> Result<super::GPUInfo, super::GPUError> {
	let vendor = super::udev_dev::get_vendor(&dev.card_node).await;

	let boot_display = device_is_boot_display(&dev.card_node)?;

	Ok(
		super::GPUInfo {
			boot_display:	boot_display,
			vendor:		vendor,
			nodes:		dev,
		}
	)
}

/*
	Check if a device is connected to boot display.
	There is a udev attribute boot_display implemented in
		https://github.com/torvalds/linux/blob/master/drivers/gpu/drm/drm_sysfs.c
	Which can be more reliable and faster than querying device connector status ourselves.
	The attribute is exported within DEVPATH/boot_display when true

	The error needs to be handled gracefully.

	We will see whether this wakes up discrete GPU, if it does, then read the file manually
*/
fn device_is_boot_display(card_device: &udev::Device) -> Result<bool, super::GPUError> {
	let boot_display_attr_value = card_device.attribute_value("boot_display");
	match boot_display_attr_value {
		Some(v)	=> {
			match v.to_str() {
				Some("1")	=> {
					return Ok(true);
				}
				Some("0")	=> {
					return Ok(false);
				}
				Some(v)	=> {
					return Err(
						super::GPUError::InvalidBootDisplay(format!("{v:?}"))
					)
				}
				None	=> {
					return Err(
						super::GPUError::InvalidBootDisplay(
							format!("Non-UTF-8"),
						)
					)
				}
			}
		}
		None	=> {}
	};

	let parent_device = {
		let parent_device = card_device.parent();
		match parent_device {
			Some(v)	=> {v}
			None	=> {
				return Ok(false);
			}
		}
	};

	let boot_vga = match parent_device.attribute_value("boot_vga") {
		Some(v)	=> v.to_str(),
		None	=> return Ok(false),
	};

	match boot_vga {
		Some("1")	=> Ok(true),
		Some("0")	=> Ok(false),
		Some(v)	=> {
			Err(
				super::GPUError::InvalidBootVGA(format!("{v:?}"))
			)
		}
		None	=> {
			Ok(false)
		}
	}
}
