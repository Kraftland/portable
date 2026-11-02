use crate::bind::subsystems::devices::gpu_ng::enumerate::EnumerateError;

/**
	Associate a random list of devices to a pair of render and card GPU device node.

	This may not be particularly fast, but considering ordinary desktop may have at most
	4 nodes, it should be fine.

	We see the card node as master, and match renderer node for it.

	If it does not match, then we can return an error.

	The internal matching logic uses DEVPATH, because ID_PATH may be systemd-udev specific.

	By removing the final component of PathBuf, we can confirm whether they share a parent.
*/
pub fn associate(devices: Vec<udev::Device>) -> Result<Vec<super::GPUPair>, EnumerateError> {
	let (cards, renderers) = {
		let mut cards = vec![];
		let mut renderers = vec![];

		for dev in devices {
			let mut devpath: std::path::PathBuf = dev.devpath().into();

			let devpath = devpath.parent().unwrap_or(&devpath);

			let node_type = match dev.sysname().to_str() {
				Some(v)	=> {
					if v.starts_with("card") {
						NodeType::Card
					} else if v.starts_with("render") {
						NodeType::Renderer
					} else {
						continue;
					}
				}
				None	=> {
					use crate::bind::subsystems::devices::gpu_ng::enumerate::EnumerateError;

					return Err(
						EnumerateError::NonUnicode(dev.sysname().to_owned())
					);
				}
			};

			match node_type {
				NodeType::Card		=> {
					cards.push(
						GPUWithDevPath {
							device:			dev,
							devpath_stripped:	devpath.to_path_buf(),
						}
					);
				}
				NodeType::Renderer	=> {
					renderers.push(
						GPUWithDevPath {
							device:			dev,
							devpath_stripped:	devpath.to_path_buf(),
						}
					);
				}
			}
		};
		(cards, renderers)
	};

	let mut renderers: std::collections::HashMap<_,_> = renderers
		.into_iter()
		.map(|f| (f.devpath_stripped, f.device))
		.collect();

	let mut ret = vec![];

	for card in cards {
		if let Some(v) = renderers.remove(&card.devpath_stripped) {
			ret.push(
				super::GPUPair {
					parent:		v.parent(),
					renderer:	v,
					card:		card.device,
				},
			);
		} else {
			return Err(
				EnumerateError::RendererMissing(card.device.sysname().into())
			);
		}
	};

	Ok(ret)
}

struct GPUWithDevPath {
	/**
		udev device, could be either render or card
	*/
	device:			udev::Device,
	/**
		Devpath with the final component stripped. aka. Parent.
	*/
	devpath_stripped:	std::path::PathBuf,
}

enum NodeType {
	Card,
	Renderer,
}
