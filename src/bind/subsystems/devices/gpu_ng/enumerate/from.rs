mod associate;
mod devlink;
mod boot_display;

/// GPUPair describes a pair of render node and card node in udev.
pub struct GPUPair {
	pub renderer:	udev::Device,
	pub card:	udev::Device,
	pub parent:	Option<udev::Device>,
}

/**
	Enumerate the list of GPUs and populate GPUInfo.

	This function may fail and it should be tolerated as None.
*/
pub async fn enumerate() -> Result<Vec<super::GPUInfo>, super::EnumerateError> {
	let devices = crate::bind::subsystems::devices::enumerate(
		crate::bind::subsystems::devices::Filter::SubsystemWithDevtype {
			subsystem:	"drm".into(),
			devtype:	"drm_minor".into(),
		}
	)
		.map_err(super::EnumerateError::UdevEnumerateError)
		?;

	let devices = associate::associate(devices)?;

	let mut ret = vec![];

	for device in devices {
		let mut boot = boot_display::device_is_boot_display(&device.card)?;

		if ! boot {
			match &device.parent {
				Some(v)	=> {
					boot = boot_display::device_is_boot_display(&v)?;
				}
				None	=> {}
			}
		}

		ret.push(
			super::GPUInfo {
				boot_display:	boot,
				vendor:		{
					match &device.parent {
						Some(v)	=> v.into(),
						None	=> {
							(&device.card).into()
						}
					}
				},

				render_name:	device.renderer.sysname().to_string_lossy().into(),
				render_devlink:	devlink::get(&device.renderer)?,
				render_devnode:	{
					match device.renderer.devnode() {
						Some(v)	=> Some(v.to_path_buf()),
						None	=> None,
					}
				},
				render_syspath:	 device.renderer.syspath().to_path_buf(),

				card_name:	device.card.sysname().to_string_lossy().into(),
				card_devlink:	devlink::get(&device.card)?,
				card_devnode:	{
					match device.card.devnode() {
						Some(v)	=> Some(v.to_path_buf()),
						None	=> None,
					}
				},
				card_syspath:	device.card.syspath().to_path_buf(),

				parent_devlink:	{
					match &device.parent {
						Some(v)	=> {
							devlink::get(v)?
						}
						None	=> {vec![]}
					}
				},
				parent_devnode:	{
					match &device.parent {
						Some(v)	=> {
							v.devnode().map(|x| x.to_path_buf())
						}
						None	=> {None}
					}
				},
				parent_syspath:	device.parent.map(|x| x.syspath().to_path_buf()),
			}
		);
	};

	unimplemented!()


}
