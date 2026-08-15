/**
	Eumerates all graphics cards (and renderer nodes, paired together) as vectors of udev devices
	See GPUDevice struct for more details
	Errors needs to be handled gracefully.
*/
pub async fn enumerate(
	logger:	&tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
) -> Result<Vec<super::GPUInfo>, super::GPUError> {
	let devices = crate::bind::subsystems::devices::enumerate(
		crate::bind::subsystems::devices::Filter::SubsystemWithDevtype {
			subsystem: "drm".to_string(),
			devtype: "drm_minor".to_string(),
		},
	)
		.await
		.map_err(super::GPUError::Enumerate)
		?;

	let _ = logger.send(
		crate::logger::LogMessage {
			level: crate::logger::LogLevel::Debug,
			message: format!("Enumerated {} cards and renderers", devices.len()),
		}
	).await;
}
