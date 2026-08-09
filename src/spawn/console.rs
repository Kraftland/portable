
/*
	The PtyPair struct describes a PTY file descriptor pair. Of which contains master and slave.
	The Master descriptor is intended to be used by daemon, while slave is used by Init (PID 1),
	Master descriptor is capable of resizing and stuff, but needs to handle manually.
*/
pub struct PtyPair {
	pub master:		nix::pty::PtyMaster,
	pub slave:		std::os::fd::OwnedFd,
	pub slave_name:		PtsName,
}

pub type PtsName = String;

#[derive(thiserror::Error, Debug)]
pub enum PtyError {
	#[error("Could not allocate new pty pair: {0:#?}")]
	NewPtyError(nix::Error),
}

impl PtyPair {
	async fn new() -> Result<Self, PtyError> {
		let pair = nix::pty::openpty(None, None)
			.map_err(PtyError::NewPtyError)
			?;

		// Scary!
		let master = unsafe {
			nix::pty::PtyMaster::from_owned_fd(pair.master)
		};

		Ok(
			PtyPair {
				slave_name:	nix::pty::ptsname_r(&master)
							.map_err(PtyError::NewPtyError)?,
				master:		master,
				slave:		pair.slave,
			},
		)
	}
}
