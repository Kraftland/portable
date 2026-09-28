#[derive(thiserror::Error, Debug)]
pub enum KvmError {
	#[error("Could not enumerate KVM devices: {0}")]
	ProbeError(std::io::Error),
}

use crate::bind::types::BindRules;

pub async fn scan() -> Result<BindRules, KvmError> {
	let mut enumerator = {
		udev::Enumerator::new()
			.map_err(KvmError::ProbeError)
			?
	};

	enumerator
		.match_sysname("kvm")
		.map_err(KvmError::ProbeError)
		?;

	let devices = enumerator.scan_devices()
		.map_err(KvmError::ProbeError)
		?;

	let mut ret = vec![];

	for dev in devices {
		ret.extend(
			super::bind_udev_device(&dev).await
		);
	};

	use crate::bind::types::DeDupRules;
	Ok(DeDupRules::dedup(ret))
}
