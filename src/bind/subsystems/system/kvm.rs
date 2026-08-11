/**
	This function mounts /dev/kvm if config allows such operation
*/
pub async fn mount_kvm(
	device_allow: Vec<crate::config::config_definition::DeviceAllow>
) -> Result<Vec<crate::bind::types::BindRule>, KvmError> {
	use crate::bind::types::BindRule;
	if ! allow_kvm(device_allow) {
		return Ok(vec![]);
	};

	let mut ret = vec![];

	if tokio::fs::try_exists("/dev/kvm").await.map_err(KvmError::IOError)? {
		ret.push(
			BindRule::Path {
				source: "/dev/kvm".into(),
				dest: "/dev/kvm".into(),
				class: crate::bind::types::BindType::Device,
			}
		);
	};

	Ok(ret)
}

fn allow_kvm(device_allow: Vec<crate::config::config_definition::DeviceAllow>) -> bool {
	for allow in device_allow {
		match allow {
			crate::config::config_definition::DeviceAllow::Kvm	=> {return true;}
			_							=> {continue;}
		}
	};
	false
}

#[derive(thiserror::Error, Debug)]
pub enum KvmError {
	#[error("I/O error probing KVM support: {0:#?}")]
	IOError(std::io::Error),
}
