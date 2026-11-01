/**
	Bind /dev/kfd for ROCm
*/
pub async fn bind_rocm_device() -> Result<crate::bind::types::BindRules, super::VendorError> {
	let kfd_path: std::path::PathBuf = ["/dev", "kfd"].iter().collect();

	match tokio::fs::try_exists(&kfd_path).await {
		Ok(true)	=> {
			Ok(
				vec![
					crate::bind::types::BindRule::Path {
						source:	kfd_path.to_path_buf(),
						dest:	kfd_path,
						class:	crate::bind::types::BindType::Device,
					}
				]
			)
		}
		Ok(false)	=> {
			Ok(vec![])
		}
		Err(e)		=> {
			Err(super::VendorError::KfdIOError(e))
		}
	}
}
