use super::GPUVendor;
use super::nvidia;

pub async fn prime_offload_envs(
	vendor: &GPUVendor,
	env_tx: &crate::envs::holder::HoldChannel,
) {
	match vendor {
		GPUVendor::NVIDIA { driver }	=> {
			match driver {
				nvidia::NVIDIADriver::Nouveau			=> {
					env_tx.send(
						crate::envs::holder::EnvMessage::Add {
							key: "DRI_PRIME".into(),
							value: "1".into(),
						},
					).await.expect("Could not set offload envs");
				}
				nvidia::NVIDIADriver::Proprietary		=> {
					env_tx.send(
						crate::envs::holder::EnvMessage::Add {
							key: "__NV_PRIME_RENDER_OFFLOAD".into(),
							value: "1".into(),
						},
					).await.expect("Could not set offload envs");
					env_tx.send(
						crate::envs::holder::EnvMessage::Add {
							key: "__VK_LAYER_NV_optimus".into(),
							value: "NVIDIA_only".into(),
						},
					).await.expect("Could not set offload envs");
					env_tx.send(
						crate::envs::holder::EnvMessage::Add {
							key: "__GLX_VENDOR_LIBRARY_NAME".into(),
							value: "nvidia".into(),
						},
					).await.expect("Could not set offload envs");
					env_tx.send(
						crate::envs::holder::EnvMessage::Add {
							key: "VK_LOADER_DRIVERS_SELECT".into(),
							value: "nvidia_icd.json".into(),
						},
					).await.expect("Could not set offload envs");
				}
				nvidia::NVIDIADriver::Unknown { driver: _ }	=> {

				}
			}
		}
		_				=> {
			env_tx.send(
				crate::envs::holder::EnvMessage::Add {
					key: "DRI_PRIME".into(),
					value: "1".into(),
				},
			).await.expect("Could not set offload envs");
		}
	}
}
