/**
	Eumerates all graphics cards (and renderer nodes, paired together) as vectors of udev devices
	See GPUDevice struct for more details
	Errors needs to be handled gracefully.
*/
pub async fn enumerate(
	logger:	&crate::logger::LogSender,
) -> Result<Vec<super::GPUInfo>, super::GPUError> {
	let devices = crate::bind::subsystems::devices::enumerate(
		crate::bind::subsystems::devices::Filter::SubsystemWithDevtype {
			subsystem:	"drm".to_string(),
			devtype:	"drm_minor".to_string(),
		},
	)
		.await
		.map_err(super::GPUError::Enumerate)
		?;

	let gpus = super::associate::associate(devices, &logger).await;

	let mut info_workers = vec![];

	for gpu in gpus {
		info_workers.push(
			tokio::spawn(
				super::get_info::get(
					gpu,
					logger.clone(),
				)
			)
		);
	};

	let mut ret = vec![];

	for worker in info_workers {
		ret.push(
			worker
				.await
				.map_err(super::GPUError::Spawn)
				?
				?
		);
	};

	Ok(ret)
}
