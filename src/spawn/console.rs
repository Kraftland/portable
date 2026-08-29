
/*
	The PtyPair struct describes a PTY file descriptor pair. Of which contains master and slave.
	The Master descriptor is intended to be used by daemon, while slave is used by Init (PID 1),
	Master descriptor is capable of resizing and stuff, but needs to handle manually.
*/
pub struct PtyPair {
	pub master:		std::os::fd::OwnedFd,
	pub slave:		std::os::fd::OwnedFd,
}

pub type PtsName = String;

#[derive(thiserror::Error, Debug)]
pub enum PtyError {
	#[error("Could not allocate new pty pair: {0:#?}")]
	NewPtyError(nix::Error),
}

impl PtyPair {
	pub fn new(columns: u16, rows: u16) -> Result<Self, PtyError> {
		let winsize = nix::pty::Winsize {
			ws_row:		rows,
			ws_col:		columns,
			ws_xpixel:	0,
			ws_ypixel:	0,
		};
		let pair = nix::pty::openpty(Some(&winsize), None)
			.map_err(PtyError::NewPtyError)
			?;

		Ok(
			PtyPair {
				master:		pair.master,
				slave:		pair.slave,
			},
		)
	}
}
