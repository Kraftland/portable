/**
	Generate a set of bind rules for a specific GPU
*/
pub async fn generate_bind_rules(
	gpu:		super::GPUInfo,
	logger:		&tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
) -> Vec<crate::bind::types::BindRule> {
	use crate::bind::types::BindRule;
	use super::GPUVendor;
	let (tx, mut rx) = tokio::sync::mpsc::unbounded_channel();

	/*
		sys/class/drm/ does not seem to be covered by udev
	*/

	{
		let drm_path = std::path::PathBuf::from("/sys/class/drm");
		match gpu.nodes.card_node {
			Some(ref v)	=> {
				let mut card_path = drm_path.clone();
				card_path.push(v.sysname());
				tx.send(
					BindRule::Path {
						source: card_path.clone(),
						dest: card_path,
						class: crate::bind::types::BindType::Device,
					},
				).expect("Error sending drm bind rules");
			}
			None	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("Missing card node: {:#?}", gpu),
					},
				).await;
			}
		};
		match gpu.nodes.render_node {
			Some(ref v)	=> {
				let mut card_path = drm_path.clone();
				card_path.push(v.sysname());
				tx.send(
					BindRule::Path {
						source: card_path.clone(),
						dest: card_path,
						class: crate::bind::types::BindType::Device,
					},
				).expect("Error sending drm bind rules");
			}
			None	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("Missing renderer node: {:#?}", gpu),
					},
				).await;
			}
		}
	};

	match gpu.vendor {
		GPUVendor::AMD			=> {
			tx.send(
				BindRule::Path {
					source: std::path::PathBuf::from("/dev/kfd"),
					dest: std::path::PathBuf::from("/dev/kfd"),
					class: crate::bind::types::BindType::Device,
				},
			).unwrap();
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
						tx.send(
							BindRule::Path {
								source: path.to_path_buf(),
								dest: path.to_path_buf(),
								class: crate::bind::types::BindType::Device,
							}
						).expect("Could not send rules for NVIDIA device");
					};
				}
				super::nvidia::NVIDIADriver::Unknown { driver }	=> {
					let _ = logger.send(
						crate::logger::LogMessage {
							level: crate::logger::LogLevel::Warn,
							message: format!("Unknown nvidia driver: {driver:?}"),
						},
					).await;
				}
			}

			let nv_modules_mount = tokio::task::spawn_blocking(|| {
				super::nvidia::nvidia_module_mounts(false)
			});

			let nv_modules_mount = match nv_modules_mount.await {
				Ok(v)	=> {v}
				Err(e)	=> {
					let _ = logger.send(
						crate::logger::LogMessage {
							level: crate::logger::LogLevel::Warn,
							message: format!(
								"Could not apply NVIDIA quirks: {:#?}",
								e,
							),
						}
					).await;
					return vec![];
				}
			};
			for rule in nv_modules_mount {
				tx.send(rule).unwrap();
			};
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

	rx.close();

	let mut ret = vec![];

	loop {
		let msg = rx.recv().await;
		match msg {
			Some(v)	=> {
				ret.push(v);
			}
			None	=> {
				break;
			}
		}
	}
	ret
}
