/**
	Finds a Wayland socket according to spec:

	[FDO](https://wayland.freedesktop.org/docs/html/apb.html#:~:text=wl%5Fdisplay%5Fconnect%20%2D%20Connect%20to%20a%20Wayland%20display%2E)

	WAYLAND_SOCKET is unsupported because we cannot pass descriptors elegantly via systemd
*/
pub async fn find(
	runtime_dir: std::path::PathBuf,
) -> Result<std::path::PathBuf, super::DisplayBindError> {
	use crate::bind::display::exists;

	/*
		use WAYLAND_DISPLAY environment variable if it is set
		otherwise display "wayland-0" will be used
	*/
	let display_name = std::env::var("WAYLAND_DISPLAY").unwrap_or("wayland-0".into());
	let display_path = std::path::PathBuf::from(display_name);

	/*
		If name is a relative path,
		then the socket is opened relative to the XDG_RUNTIME_DIR directory.

		If name is an absolute path,
		then that path is used as-is for the location of the socket
		at which the Wayland server is listening;
		no qualification inside XDG_RUNTIME_DIR is attempted.
	*/
	if ! display_path.is_absolute() {
		let mut path = runtime_dir;
		path.push(display_path);
		match exists(path.clone()).await.map_err(super::DisplayBindError::IOError)? {
			true	=> {
				Ok(path)
			}
			false	=> {
				Err(super::DisplayBindError::NonExistentError)
			}
		}
	} else {
		match exists(display_path.clone()).await.map_err(super::DisplayBindError::IOError)? {
			true	=> {
				Ok(display_path)
			}
			false	=> {
				Err(super::DisplayBindError::NonExistentError)
			}
		}
	}
}
