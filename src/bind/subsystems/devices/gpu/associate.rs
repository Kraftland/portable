/**
	Associates the card device with renderer, using the GPUDevice struct above
	Internally uses the ID_PATH approach just like the previous impl
*/
pub async fn associate(
	devices:	Vec<udev::Device>,
	logger:		&crate::logger::LogSender,
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
					crate::logger::messages::LogMessage::Warn(
						crate::logger::messages::warn::WarnMessage::MissingIDPath(
							device.sysname().into(),
						)
					)
				).await;
				continue;
			}
		};

		let dev_type = match card_type(&device) {
			Some(v)	=> v,
			None	=> {
				let _ = logger.send(
					crate::logger::messages::LogMessage::Warn(
						crate::logger::messages::warn::WarnMessage::UnknownDeviceGPUType(
							device.sysname().into(),
						)
					)
				).await;
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
					crate::logger::messages::LogMessage::Warn(
						crate::logger::messages::warn::WarnMessage::CardRendererNodeMissing(k.into())
					)
				).await;
				continue;
			}
		};

		let renderer = match v.1 {
			Some(v)	=> v,
			None	=> {
				let _ = logger.send(
					crate::logger::messages::LogMessage::Warn(
						crate::logger::messages::warn::WarnMessage::CardRendererNodeMissing(k.into())
					)
				).await;
				continue;
			}
		};

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
