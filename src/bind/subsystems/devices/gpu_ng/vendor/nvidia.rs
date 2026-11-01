#[derive(PartialEq, Eq)]
pub enum NVIDIADriver {
	Proprietary,
	Nouveau,
}

/*
	Scan /dev/ for nvidia device nodes that aren't in udev database.

	It should only be invoked when NVIDIADriver::Proprietary, as nouveau does not have these
	issues.
*/
pub async fn get_nvidia_devices()
->
	Result<Vec<std::path::PathBuf>, crate::bind::subsystems::devices::gpu_ng::vendor::VendorError>
{
	use crate::bind::subsystems::devices::gpu_ng::vendor::VendorError;

	let mut dir = tokio::fs::read_dir("/dev")
		.await
		.map_err(VendorError::ScanNvidiaDevError)
		?;

	let mut ret = vec![];

	while let Some(v) = dir.next_entry().await.map_err(VendorError::ScanNvidiaDevError)? {
		match v.file_name().to_str() {
			Some(name)	=> {
				if name.starts_with("nvidia") {
					ret.push(v.path());
				}
			}
			None	=> {}
		}
	}
	Ok(ret)
}
