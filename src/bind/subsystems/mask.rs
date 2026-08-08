mod paths;

/**
	The public function is used to mask certain paths for privacy
*/
pub async fn mask_paths() -> Result<crate::bind::types::BindRules, MaskError> {
	let mut ret = vec![];
	use crate::bind::types::BindRule;

	let mut status = vec![];

	for path in paths::get_paths() {
		status.push(
			(path.clone(), tokio::fs::try_exists(path))
		);
	};

	for entry in status {
		if entry.1.await.map_err(MaskError::ExistError)? {
			ret.push(
				BindRule::VirtualFS {
					dest: entry.0,
					class: crate::bind::types::VirtualFS::Tmpfs {
						size_mb: Some(0),
						perms: None,
					},
				},
			);
		}
	};

	Ok(ret)
}

#[derive(thiserror::Error, Debug)]
pub enum MaskError {
	#[error("Could not determine if path exists: {0:#?}")]
	ExistError(std::io::Error),

	#[error("Could not determine if path exists: {0:#?}")]
	SpawnError(tokio::task::JoinError),
}
