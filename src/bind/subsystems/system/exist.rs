/**
	This public function list_exists is designed to run on a blocking tokio thread and batch
	try_exists calls in one.

	It may yield significant time saving compared to looping tokio::fs::try_exists, since
	it avoids async scheduling overhead.

	It should be known that permission error could cause this function to fail.
*/
pub async fn list_exists<T, P>(paths: T) -> Result<Vec<std::path::PathBuf>, ExistError>
where
	T:	IntoIterator<Item = P> + Send + 'static,
	P:	AsRef<std::path::Path>,
{
	tokio::task::spawn_blocking(
		|| {
			let mut ret = vec![];
			for path in paths {
				if
					std::fs::exists(path.as_ref())
						.map_err(ExistError::IOError)
						?
				{
					ret.push(path.as_ref().to_path_buf());
				}
			};

			Ok(ret)
		}
	)
		.await
		.map_err(ExistError::SpawnError)
		?
}

#[derive(Debug, thiserror::Error)]
pub enum ExistError {
	#[error("Could not spawn blocking thread: {0}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not determine if path exists: {0}")]
	IOError(std::io::Error),
}
