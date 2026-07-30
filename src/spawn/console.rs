
/*
	The PtyPair struct describes a PTY file descriptor pair. Of which contains master and slave.
	The Master descriptor is intended to be used by daemon, while slave is used by Init (PID 1),
	Master descriptor is capable of resizing and stuff, but needs to handle manually.
*/
pub struct PtyPair {
	master:		std::os::fd::OwnedFd,
	slave:		std::os::fd::OwnedFd,
}

#[derive(thiserror::Error, Debug)]
pub enum PtyError {
	#[error("Could not allocate new pty pair: {0:#?}")]
	NewPtyError(nix::Error),
}

impl PtyPair {
	async fn new() -> Result<Self, PtyError> {
		let pair = nix::pty::openpty(None, None)
			.map_err(PtyError::NewPtyError)?;
		Ok(
			PtyPair {
				master: pair.master,
				slave: pair.slave,
			},
		)
	}
}
