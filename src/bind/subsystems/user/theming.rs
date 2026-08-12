pub async fn bind(
	translator:	crate::bind::translate::Delta,
	xdg_config:	std::sync::Arc<std::path::PathBuf>,
) -> Result<crate::bind::types::BindRules, super::UserBindError> {
	let mut ret = vec![];

	for path in paths(translator, xdg_config).await? {
		// Returns true if the path exists on disk and is pointing at a directory
		if path.0.is_dir() {
			ret.push(
				crate::bind::types::BindRule::Path {
					source: path.0,
					dest: path.1,
					class: crate::bind::types::BindType::ReadOnly,
				}
			);
		}
	}

	Ok(ret)
}

/**
	Returns a list of (source, dest) PathBufs to bind
*/
async fn paths(
	translator:	crate::bind::translate::Delta,
	xdg_config:	std::sync::Arc<std::path::PathBuf>,
) -> Result<Vec<(std::path::PathBuf, std::path::PathBuf)>, super::UserBindError> {
	use crate::bind::translate::Translate;
	let fontconfig_host = {
		let mut path = xdg_config.to_path_buf();
		path.push("fontconfig");
		path
	};
	let fontconfig_nested = fontconfig_host
		.translate_home(&translator)
		.await
		.map_err(super::UserBindError::TranslatePathError)
		?;

	let (gtk3_css_host, gtk3_css_nested) = {
		let mut path = xdg_config.to_path_buf();
		path.push("gtk-3.0");
		path.push("gtk.css");

		let nested = path
			.translate_home(&translator)
			.await
			.map_err(super::UserBindError::TranslatePathError)
			?;

		(path, nested)
	};

	let (gtk3_noctalia_host, gtk3_noctalia_nested) = {
		let mut path = xdg_config.to_path_buf();
		path.push("gtk-3.0");
		path.push("noctalia.css");

		let nested = path
			.translate_home(&translator)
			.await
			.map_err(super::UserBindError::TranslatePathError)
			?;

		(path, nested)
	};

	Ok(vec![
		(fontconfig_host, fontconfig_nested),
		(gtk3_css_host, gtk3_css_nested),
		(gtk3_noctalia_host, gtk3_noctalia_nested),
	])
}
