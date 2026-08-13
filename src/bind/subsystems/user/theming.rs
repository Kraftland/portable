pub async fn bind(
	translator:	crate::bind::translate::Delta,
	xdg_config:	std::path::PathBuf,
	xdg_data:	std::path::PathBuf,
) -> Result<crate::bind::types::BindRules, super::UserBindError> {
	let mut ret = vec![];

	for path in paths(translator, xdg_config, xdg_data).await? {
		// Returns true if the path exists on disk and is pointing at a directory
		if path.0.is_dir() {
			ret.push(
				crate::bind::types::BindRule::Path {
					source: path.0,
					dest: path.1,
					class: crate::bind::types::BindType::ReadOnly,
				}
			);
		} else if path.0.is_file() {
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
	xdg_config:	std::path::PathBuf,
	xdg_data:	std::path::PathBuf,
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

	let (gtk3_colours_host, gtk3_colours_nested) = {
		let mut path = xdg_config.to_path_buf();
		path.push("gtk-3.0");
		path.push("colors.css");

		let nested = path
			.translate_home(&translator)
			.await
			.map_err(super::UserBindError::TranslatePathError)
			?;

		(path, nested)
	};

	let (gtk4_css_host, gtk4_css_nested) = {
		let mut path = xdg_config.to_path_buf();
		path.push("gtk-4.0");
		path.push("gtk.css");

		let nested = path
			.translate_home(&translator)
			.await
			.map_err(super::UserBindError::TranslatePathError)
			?;

		(path, nested)
	};

	let (kdeglobals_host, kdeglobals_nested) = {
		let mut path = xdg_config.to_path_buf();
		path.push("kdeglobals");

		let nested = path
			.translate_home(&translator)
			.await
			.map_err(super::UserBindError::TranslatePathError)
			?;

		(path, nested)
	};

	let (qt6ct_host, qt6ct_nested) = {
		let mut path = xdg_config.to_path_buf();
		path.push("qt6ct");

		let nested = path
			.translate_home(&translator)
			.await
			.map_err(super::UserBindError::TranslatePathError)
			?;

		(path, nested)
	};

	/*
		We might consider to drop this as it allows fingerprinting
	*/
	let (fonts_host, fonts_nested) = {
		let mut path = xdg_data.to_path_buf();
		path.push("fonts");

		let nested = path
			.translate_home(&translator)
			.await
			.map_err(super::UserBindError::TranslatePathError)
			?;

		(path, nested)
	};
	let (icons_host, icons_nested) = {
		let mut path = xdg_data.to_path_buf();
		path.push("icons");

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
		(gtk3_colours_host, gtk3_colours_nested),
		(gtk4_css_host, gtk4_css_nested),
		(kdeglobals_host, kdeglobals_nested),
		(qt6ct_host, qt6ct_nested),
		(fonts_host, fonts_nested),
		(icons_host, icons_nested),
	])
}
