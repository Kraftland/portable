#[derive(thiserror::Error, Debug)]
pub enum CameraError {
	#[error("Could not enumerate camera devices: {0:#?}")]
	EnumerateError(super::EnumerateError),
}

use crate::bind::types::BindRules;

async fn scan() -> Result<BindRules, CameraError> {
	let devices = super::enumerate(
		super::Filter::Subsystem {
			subsystem: "video4linux".into(),
		},
	)
		.await
		.map_err(CameraError::EnumerateError)
		?;

	let mut ret = vec![];

	for dev in devices {
		ret.extend(
			super::bind_udev_device(dev).await
		);
	};

	use crate::bind::types::DeDupRules;
	Ok(DeDupRules::dedup(ret))
}
