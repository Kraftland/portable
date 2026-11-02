mod associate;
mod devlink;

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

	unimplemented!()


}
