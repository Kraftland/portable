pub async fn share_path_with_helper(
	bus_conn:	zbus::Connection,
	directory:	bool,
) -> Result<(), ShareError> {
	unimplemented!()
}

#[derive(thiserror::Error, Debug)]
pub enum ShareError {}
