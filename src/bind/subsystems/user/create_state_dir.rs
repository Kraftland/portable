pub async fn create_state_dir(
	state_home:	&std::path::PathBuf,
) -> Result<(), super::UserBindError> {
	tokio::fs::create_dir_all(state_home)
		.await
		.map_err(super::UserBindError::CreateHomeError)
}
