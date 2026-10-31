impl From<&super::GPUInfo> for Vec<crate::bind::types::BindRule> {
	fn from(value: &super::GPUInfo) -> Self {
		let mut ret: Vec<crate::bind::types::BindRule> = vec![];

		// /sys/class/drm is not convered yet.
		{
			let drm_path: std::path::PathBuf = ["/sys", "class", "drm"].iter().collect();

			let card_path = {
				let mut path = drm_path.to_path_buf();
				path.push(value.card_name.as_str());
				path
			};
			let render_path = {
				let mut path = drm_path.to_path_buf();
				path.push(value.render_name.as_str());
				path
			};

			ret.push(
				crate::bind::types::BindRule::Path {
					source:	card_path.to_path_buf(),
					dest:	card_path,
					class:	crate::bind::types::BindType::Device,
				}
			);
			ret.push(
				crate::bind::types::BindRule::Path {
					source:	render_path.to_path_buf(),
					dest:	render_path,
					class:	crate::bind::types::BindType::Device,
				}
			);
		};

		// Basically bind_udev_device in devices
		// devlinks first
		for link in &value.card_devlink {
			ret.push(
				crate::bind::types::BindRule::Path {
					source:	link.to_path_buf(),
					dest:	link.to_path_buf(),
					class:	crate::bind::types::BindType::Device,
				}
			);
		};
		for link in &value.render_devlink {
			ret.push(
				crate::bind::types::BindRule::Path {
					source:	link.to_path_buf(),
					dest:	link.to_path_buf(),
					class:	crate::bind::types::BindType::Device,
				}
			);
		};
		for link in &value.parent_devlink {
			ret.push(
				crate::bind::types::BindRule::Path {
					source:	link.to_path_buf(),
					dest:	link.to_path_buf(),
					class:	crate::bind::types::BindType::Device,
				}
			);
		};

		// Kernel device node under /dev
		match &value.card_devnode {
			Some(v)	=> {
				ret.push(
					crate::bind::types::BindRule::Path {
						source:	v.to_path_buf(),
						dest:	v.to_path_buf(),
						class:	crate::bind::types::BindType::Device,
					}
				);
			}
			None	=> {}
		}
		match &value.render_devnode {
			Some(v)	=> {
				ret.push(
					crate::bind::types::BindRule::Path {
						source:	v.to_path_buf(),
						dest:	v.to_path_buf(),
						class:	crate::bind::types::BindType::Device,
					}
				);
			}
			None	=> {}
		}
		match &value.parent_devnode {
			Some(v)	=> {
				ret.push(
					crate::bind::types::BindRule::Path {
						source:	v.to_path_buf(),
						dest:	v.to_path_buf(),
						class:	crate::bind::types::BindType::Device,
					}
				);
			}
			None	=> {}
		}

		// Finally, syspath
		match &value.card_syspath {
			Some(v)	=> {
				ret.push(
					crate::bind::types::BindRule::Path {
						source:	v.to_path_buf(),
						dest:	v.to_path_buf(),
						class:	crate::bind::types::BindType::Device,
					}
				);
			}
			None	=> {}
		}
		match &value.render_syspath {
			Some(v)	=> {
				ret.push(
					crate::bind::types::BindRule::Path {
						source:	v.to_path_buf(),
						dest:	v.to_path_buf(),
						class:	crate::bind::types::BindType::Device,
					}
				);
			}
			None	=> {}
		}
		match &value.parent_syspath {
			Some(v)	=> {
				ret.push(
					crate::bind::types::BindRule::Path {
						source:	v.to_path_buf(),
						dest:	v.to_path_buf(),
						class:	crate::bind::types::BindType::Device,
					}
				);
			}
			None	=> {}
		}

		ret
	}
}
