mod content;
mod write;

#[derive(thiserror::Error, Debug)]
pub enum FlatpakInfoError {
	#[error("I/O error creating file: {0:#?}")]
	CreateError(std::io::Error),

	#[error("I/O error writing file: {0:#?}")]
	WriteError(std::io::Error),
}

/**
	Create the flatpak-info file
*/
pub async fn create(
	config:			std::sync::Arc<crate::config::config_definition::Config>,
	instance_id:		std::sync::Arc<String>,
	xdg:			std::sync::Arc<crate::xdg::XdgDirs>,

	portable_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,

	#[cfg(feature = "flatpak")]
	flatpak_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::flatpak::FlatpakRuntime>,
) -> Result<std::path::PathBuf, FlatpakInfoError> {
	let content_string = content::generate(config, instance_id, xdg).await;
	write::write(content_string, portable_runtime, flatpak_runtime).await
}
