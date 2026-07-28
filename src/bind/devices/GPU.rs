use thiserror::Error;

use crate::bind::types::BindRule;

#[derive(Error, Debug)]
pub enum GPUError {
	#[error("Could not determine boot display: invalid value {0:?}")]
	InvalidBootDisplay(String),
	#[error("Could not determine boot vga: invalid value {0:?}")]
	InvalidBootVGA(String),
}

// pub async fn scan(all_gpus: bool) -> Result<Vec<BindRule>, GPUError> {

// }


/*
	Check if a device is connected to boot display.
	There is a udev attribute boot_display implemented in
		https://github.com/torvalds/linux/blob/master/drivers/gpu/drm/drm_sysfs.c
	Which can be more reliable and faster than querying device connector status ourselves.
	The attribute is exported within DEVPATH/boot_display when true

	The error needs to be handled gracefully.

	We will see whether this wakes up discrete GPU, if it does, then read the file manually
*/
async fn device_is_boot_display(card_device: &udev::Device) -> Result<bool, GPUError> {
	let boot_display_attr_value = card_device.attribute_value("boot_display");
	match boot_display_attr_value {
		Some(v)	=> {
			if v == "1" {
				return Ok(true)
			} else {
				return Err(
					GPUError::InvalidBootDisplay(format!("{v:?}"))
				);
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

	match parent_device.attribute_value("boot_vga") {
		Some(v)	=> {
			if v == "1" {
				Ok(true)
			} else {
				Err(
					GPUError::InvalidBootVGA(format!("{v:?}"))
				)
			}
		}
		None	=> {
			Ok(false)
		}
	}
}
