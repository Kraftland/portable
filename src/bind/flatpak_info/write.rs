/**
	Write content into a flatpak-info file, returns the path on success.

	It also writes to various flatpak entries when the flatpak feature is enabled
*/
pub async fn write(
	content:		String,
	portable_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,

	#[cfg(feature = "flatpak")]
	flatpak_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::flatpak::FlatpakRuntime>,
) -> Result<std::path::PathBuf, super::FlatpakInfoError> {
	use crate::bind::subsystems::dirs::RuntimePathsTrait;
	use tokio::io::AsyncWriteExt;

	#[cfg(feature = "flatpak")]
	{
		let mut info_path = flatpak_runtime.path();
		info_path.push("info");
		let mut file = tokio::fs::OpenOptions::new()
			.read(false)
			.write(true)
			.create_new(true)
			.open(info_path)
			.await
			.map_err(super::FlatpakInfoError::CreateError)
			?;
		file.write(
			&content.as_bytes()
		)
			.await
			.map_err(super::FlatpakInfoError::WriteError)
			?
	};

	let info_path = {
		let mut info_path = portable_runtime.path();
		info_path.push("flatpak-info");
		info_path
	};
	let mut file = tokio::fs::OpenOptions::new()
		.read(false)
		.write(true)
		.create_new(true)
		.open(&info_path)
		.await
		.map_err(super::FlatpakInfoError::CreateError)
		?;
	file.write(&content.as_bytes())
		.await
		.map_err(super::FlatpakInfoError::WriteError)
		?;

	Ok(info_path)
}
