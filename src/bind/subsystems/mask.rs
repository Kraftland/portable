pub struct Mask {}

impl super::GenerateBind for Mask {
	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::BindError> {
		mask_paths().await
	}
	type BindError = MaskError;
}

/**
	The public function is used to mask certain paths for privacy
*/
async fn mask_paths() -> Result<crate::bind::types::BindRules, MaskError> {
	let mask_dirs = vec![
		"/proc/bus",
		"/proc/driver",

		"/sys/devices/virtual/dmi",
		"/sys/devices/virtual/block",
		"/sys/devices/virtual/sound",

		"/proc/1",
		"/usr/share/applications",
	];

	let mask_files = vec![
		// Mask certain procfs entries as they leak system info
		"/proc/uptime",
		"/proc/cmdline",
		"/proc/diskstats",
		"/proc/devices",
		"/proc/config.gz",
		"/proc/loadavg",
	];


	let mut ret = vec![];
	use crate::bind::types::BindRule;


	for path in super::system::exist::list_exists(mask_dirs)
		.await
		.map_err(MaskError::ExistError)
		?
	{
		ret.push(
			BindRule::VirtualFS {
				dest:	path,
				class:	crate::bind::types::VirtualFS::Tmpfs {
					size_mb:	Some(0),
					perms:		None,
				},
			}
		);
	};

	for path in super::system::exist::list_exists(mask_files)
		.await
		.map_err(MaskError::ExistError)
		?
	{
		ret.push(
			BindRule::Path {
				source:	"/dev/null".into(),
				dest:	path,
				class:	crate::bind::types::BindType::ReadOnly,
			}
		);
	};

	Ok(ret)
}

#[derive(thiserror::Error, Debug)]
pub enum MaskError {
	#[error("Could not determine if path exists: {0}")]
	ExistError(super::system::exist::ExistError),
}
