#[derive(PartialEq, Eq)]
pub enum NVIDIADriver {
	Proprietary,
	Nouveau,
	Unknown,
}

impl From<Option<&std::ffi::OsStr>> for NVIDIADriver {
	fn from(value: Option<&std::ffi::OsStr>) -> Self {
		let value = match value {
			Some(v)	=> v,
			None	=> return Self::Unknown,
		};

		match value.to_str() {
			Some("nvidia")	=> Self::Proprietary,
			Some("nouveau")	=> Self::Nouveau,
			_		=> Self::Unknown,
		}
	}
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

/**
	Block or mount NVIDIA module paths

	When blocked, prevents accidental discrete GPU wake up.
*/
pub async fn nvidia_module_mounts(block: bool)
	-> Result<crate::bind::types::BindRules, crate::bind::subsystems::devices::gpu_ng::vendor::VendorError>
{
	use crate::bind::subsystems::devices::gpu_ng::vendor::VendorError;
	use crate::bind::types::BindRule;
	let paths = vec![
		"/sys/module/nvidia",
		"/sys/module/nvidia_drm",
		"/sys/module/nvidia_modeset",
		"/sys/module/nvidia_uvm",
		"/sys/module/nvidia_wmi_ec_backlight",
	];

	let mut ret = vec![];

	for path in paths {
		if ! tokio::fs::try_exists(path)
			.await
			.map_err(VendorError::MountNvidiaModuleError)
			?
		{
			continue;
		}
		match block {
			true	=> {
				ret.push(
					BindRule::VirtualFS {
						dest: path.into(),
						class: crate::bind::types::VirtualFS::Tmpfs {
							size_mb: Some(0),
							perms: None,
						},
					}
				);
			}
			false	=> {
				ret.push(
					BindRule::Path {
						source: path.into(),
						dest: path.into(),
						class: crate::bind::types::BindType::Device,
					},
				);
			}
		}
	};
	Ok(ret)
}
