/**
	Associates the card device with renderer, using the GPUDevice struct above
	Internally uses the ID_PATH approach just like the previous impl
*/
pub async fn associate(
	devices:	Vec<udev::Device>,
	logger:		&tokio::sync::mpsc::Sender<crate::logger::LogMessage>,
) -> Vec<super::GPUDevice> {
	// The map is for holding (card, renderer)
	let mut map: std::collections::HashMap
		<std::ffi::OsString, (Option<udev::Device>, Option<udev::Device>)>
		= std::collections::HashMap::new();

	for device in devices {
		let id_path = match device.property_value("ID_PATH") {
			Some(v)	=> v,
			None	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!(
						"GPU {:?} does not have ID_PATH property", device.sysname(),
						)
					}
				).await;
				continue;
			}
		};

		let dev_type = match card_type(&device) {
			Some(v)	=> v,
			None	=> {
				crate::logger::LogMessage {
					level:		crate::logger::LogLevel::Warn,
					message:	format!(
						"Device {:?} has unknown type", device.sysname(),
					)
				};
				continue;
			}
		};

		let entry = map.entry(id_path.into()).or_insert((None, None));
		match dev_type {
			NodeType::Card		=> {
				entry.0 = Some(device)
			}
			NodeType::Renderer	=> {
				entry.1 = Some(device)
			}
		};
	};

	let mut ret = vec![];

	for (k, v) in map {
		let card = match v.0 {
			Some(v)	=> v,
			None	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!(
							"Could not find card node for {0:?}",
							k,
						)
					}
				).await;
				continue;
			}
		};

		let renderer = match v.1 {
			Some(v)	=> v,
			None	=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!(
							"Could not find renderer node for {0:?}",
							k,
						)
					}
				).await;
				continue;
			}
		};

		let _ = logger.send(
			crate::logger::LogMessage {
				level: crate::logger::LogLevel::Debug,
				message: format!(
					"Associated GPU card {:?} to renderer node {:?}",
					card.sysname(),
					renderer.sysname(),
				)
			}
		).await;

		ret.push(
			super::GPUDevice {
				card_node:	card,
				render_node:	renderer,
			}
		);
	};

	ret
}

enum NodeType {
	Card,
	Renderer,
}

fn card_type (device: &udev::Device) -> Option<NodeType> {
	let sys_name = device.sysname();
	let sys_name = match sys_name.to_str() {
		Some(v)	=> {v}
		None	=> {return None}
	};

	if sys_name.starts_with("card") {
		return Some(NodeType::Card)
	} else if sys_name.starts_with("render") {
		return Some(NodeType::Renderer);
	} else {
		return None;
	}
}
