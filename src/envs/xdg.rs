#[derive(Debug, thiserror::Error)]
pub enum XDGEnvError {
	#[error("Could not translate path: {0:#?}")]
	TranslateError(crate::bind::translate::TranslatePathError),
}

/**
	Generates a list of XDG environment variables
*/
pub async fn generate_xdg_envs(
	map:	&mut std::collections::HashMap<String, String>,
	xdg:	std::sync::Arc<crate::xdg::XdgDirs>,
	config:	std::sync::Arc<crate::config::Config>,
) -> Result<(), XDGEnvError> {
	use crate::bind::translate::Translate;
	let delta = crate::bind::translate::Delta::get(&config, &xdg).await;

	let sandbox_home = {
		let mut path = xdg.data_home.to_path_buf();
		path.push(&config.metadata.state_directory);
		path
	};

	{
		let xdg_config = {
			let path = xdg.config_home.to_path_buf();
			path
				.translate_home(&delta)
				.await
				.map_err(XDGEnvError::TranslateError)
				?
		};
		map.insert("XDG_CONFIG_HOME".to_string(), xdg_config.to_string_lossy().to_string());
	};

	{
		let mut docs = sandbox_home.to_path_buf();
		docs.push("Documents");

		map.insert(
			"XDG_DOCUMENTS_DIR".into(),
			docs.to_string_lossy().to_string(),
		);
	};

	{
		let mut data_home = sandbox_home.to_path_buf();
		data_home.push(".local");
		data_home.push("share");
		map.insert(
			"XDG_DATA_HOME".into(),
			data_home.to_string_lossy().to_string(),
		);
	};

	{
		let mut state_home = sandbox_home.to_path_buf();
		state_home.push(".local");
		state_home.push("state");
		map.insert(
			"XDG_STATE_HOME".into(),
			state_home.to_string_lossy().to_string(),
		);
	};

	{
		let mut cache = sandbox_home.to_path_buf();
		cache.push("cache");
		map.insert(
			"XDG_CACHE_HOME".into(),
			cache.to_string_lossy().to_string(),
		);
	};

	{
		let mut desktop = sandbox_home.to_path_buf();
		desktop.push("Desktop");
		map.insert(
			"XDG_DESKTOP_DIR".into(),
			desktop.to_string_lossy().to_string(),
		);
	};

	{
		let mut downloads = sandbox_home.to_path_buf();
		downloads.push("Downloads");
		map.insert(
			"XDG_DOWNLOAD_DIR".into(),
			downloads.to_string_lossy().to_string(),
		);
	};

	{
		let mut template = sandbox_home.to_path_buf();
		template.push("Templates");
		map.insert(
			"XDG_TEMPLATES_DIR".into(),
			template.to_string_lossy().to_string(),
		);
	};

	{
		let mut public = sandbox_home.to_path_buf();
		public.push("Public");
		map.insert(
			"XDG_PUBLICSHARE_DIR".into(),
			public.to_string_lossy().to_string(),
		);
	};

	{
		let mut desktop = sandbox_home.to_path_buf();
		desktop.push("Public");
		map.insert(
			"XDG_PUBLICSHARE_DIR".into(),
			desktop.to_string_lossy().to_string(),
		);
	};

	{
		let mut music = sandbox_home.to_path_buf();
		music.push("Music");
		map.insert(
			"XDG_MUSIC_DIR".into(),
			music.to_string_lossy().to_string(),
		);
	};

	{
		let mut pic = sandbox_home.to_path_buf();
		pic.push("Pictures");
		map.insert(
			"XDG_PICTURES_DIR".into(),
			pic.to_string_lossy().to_string(),
		);
	};

	{
		let mut videos = sandbox_home.to_path_buf();
		videos.push("Videos");
		map.insert(
			"XDG_VIDEOS_DIR".into(),
			videos.to_string_lossy().to_string(),
		);
	};

	{
		map.insert(
			"XDG_RUNTIME_DIR".into(),
			xdg.runtime.to_string_lossy().to_string(),
		);
	};

	Ok(())
}
