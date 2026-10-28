impl TryFrom<super::GPUInfo> for Vec<crate::bind::types::BindRule> {
	fn try_from(value: super::GPUInfo) -> Result<Self, Self::Error> {
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
					class:	crate::bind::types::BindType::Device }
			);
			ret.push(
				crate::bind::types::BindRule::Path {
					source:	render_path.to_path_buf(),
					dest:	render_path,
					class:	crate::bind::types::BindType::Device }
			);
		};

		unimplemented!()

	}

	type Error = GPUBindConvertError;
}

#[derive(thiserror::Error, Debug)]
pub enum GPUBindConvertError {}
