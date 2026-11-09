impl TryFrom<&super::GPUInfo> for Vec<crate::bind::types::BindRule> {
	fn try_from(value: &super::GPUInfo) -> Result<Self, Self::Error> {
		let mut ret: Vec<crate::bind::types::BindRule> = vec![];

		// /sys/class/drm is not convered yet.
		{
			let drm_path: std::path::PathBuf = ["/sys", "class", "drm"].iter().collect();

			let (card_bind_src, card_drm_path) = {
				let mut path = drm_path.to_path_buf();
				path.push(value.card_name.as_str());
				(
					std::fs::canonicalize(&path)
						.map_err(ToBindError::DrmCanonicaliseError)
						?,
					path,
				)
			};

			ret.push(
				crate::bind::types::BindRule::Symlink {
					source:	card_bind_src,
					dest:	card_drm_path,
				}
			);

			let (render_bind_src, render_drm_path) = {
				let mut path = drm_path.to_path_buf();
				path.push(value.render_name.as_str());
				(
					std::fs::canonicalize(&path)
						.map_err(ToBindError::DrmCanonicaliseError)
						?,
					path,
				)
			};

			ret.push(
				crate::bind::types::BindRule::Symlink {
					source:	render_bind_src,
					dest:	render_drm_path,
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
		ret.push(
			crate::bind::types::BindRule::Path {
				source:	value.card_syspath.to_path_buf(),
				dest:	value.card_syspath.to_path_buf(),
				class:	crate::bind::types::BindType::Device,
			}
		);
		ret.push(
			crate::bind::types::BindRule::Path {
				source:	value.render_syspath.to_path_buf(),
				dest:	value.render_syspath.to_path_buf(),
				class:	crate::bind::types::BindType::Device,
			}
		);
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

		Ok(ret)
	}

	type Error = ToBindError;
}

#[derive(Debug, thiserror::Error)]
pub enum ToBindError {
	#[error("Could not canonicalise DRM path: {0}")]
	DrmCanonicaliseError(std::io::Error),
}
