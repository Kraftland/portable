
/**
	Opens sandbox home under XDG_DATA_HOME/state_dir
*/
pub async fn open(
	xdg:		std::sync::Arc<crate::xdg::XdgDirs>,
	state_dir:	&str,
	bus:		&zbus::Connection,
) -> zbus::Result<()> {
	let path = {
		let mut path = std::path::PathBuf::from(xdg.data_home.as_path());
		path.push(state_dir);
		path
	};

	crate::ipc::portals::open_uri::open_directory(
		bus,
		None,
		path,
	)
		.await
}
