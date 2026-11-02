use crate::bind::subsystems::devices::gpu_ng::enumerate::EnumerateError;

/**
	This public function takes a borrowed udev master device and attempts to determine
	whether it is used to render the initial framebuffer employing the boot_display and
	alternatively boot_vga attribute.

	Caller should tolerate errors as false.

	Caller should re-try with parent PCIe device when false.
*/
pub fn device_is_boot_display(
	card_device:	&udev::Device,
) -> Result<bool, EnumerateError> {
	let boot_display_attr_value = card_device.attribute_value("boot_display");
	match boot_display_attr_value {
		Some(attr)	=> {
			match attr.to_str() {
				Some("1")	=> {
					return Ok(true);
				}
				Some("0")	=> {
					return Ok(false);
				}
				Some(_) | None	=> {
					return Err(
						EnumerateError::BootDisplayUnexpected(attr.into())
					)
				}
			}
		}
		None	=> {}
	};

	let boot_vga_attr_value = card_device.attribute_value("boot_vga");
	match boot_vga_attr_value {
		Some(attr)	=> {
			match attr.to_str() {
				Some("1")	=> {
					return Ok(true);
				}
				Some("0")	=> {
					return Ok(false);
				}
				Some(_) | None	=> {
					return Err(
						EnumerateError::BootVGAUnexpected(attr.into())
					)
				}
			}
		}
		None	=> {}
	};

	Ok(false)
}
