/**
	Generate a set of bind rules for a specific GPU
*/
pub async fn generate_bind_rules(
	gpu:		super::GPUInfo,
	logger:		tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
) -> Vec<crate::bind::types::BindRule> {
	use crate::bind::types::BindRule;
	use super::GPUVendor;

	let mut ret = vec![];

	/*
		sys/class/drm/ does not seem to be covered by udev
	*/

	{
		let drm_path = std::path::PathBuf::from("/sys/class/drm");

		{
			let mut card_path = drm_path.to_path_buf();
			card_path.push(gpu.nodes.card_node.sysname());

			ret.push(
				BindRule::Path {
					source: card_path.to_path_buf(),
					dest: card_path,
					class: crate::bind::types::BindType::Device,
				}
			);
		};

		{
			let mut renderer_path = drm_path.to_path_buf();
			renderer_path.push(gpu.nodes.render_node.sysname());

			ret.push(
				BindRule::Path {
					source: renderer_path.to_path_buf(),
					dest: renderer_path,
					class: crate::bind::types::BindType::Device,
				}
			);
		};
	};

	match gpu.vendor {
		GPUVendor::AMD			=> {
			ret.push(
				BindRule::Path {
					source: std::path::PathBuf::from("/dev/kfd"),
					dest: std::path::PathBuf::from("/dev/kfd"),
					class: crate::bind::types::BindType::Device,
				},
			);
		}

		GPUVendor::Intel		=> {
			// Intel doesn't need any new tricks, yet
		}
		GPUVendor::NVIDIA {driver}	=> {
			match driver {
				super::nvidia::NVIDIADriver::Nouveau			=> {
					// TODO: what about nouveau's modules?
				}
				super::nvidia::NVIDIADriver::Proprietary		=> {
					for path in super::nvidia::get_nvidia_devices(&logger) {
						ret.push(
							BindRule::Path {
								source: path.to_path_buf(),
								dest: path.to_path_buf(),
								class: crate::bind::types::BindType::Device,
							}
						);
					};
				}
				super::nvidia::NVIDIADriver::Unknown { driver }	=> {
					let _ = logger.send(
						crate::logger::LogMessage {
							level: crate::logger::LogLevel::Warn,
							message: format!(
								"Unknown nvidia driver: {driver:?}",
							),
						},
					).await;
				}
			}

			ret.extend(
				super::nvidia::nvidia_module_mounts(false).await
			);
		}

		GPUVendor::Others	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level: crate::logger::LogLevel::Warn,
					message: format!("Could not apply GPU quirks: unknown vendor"),
				}
			).await;
		}
	};
	ret
}
