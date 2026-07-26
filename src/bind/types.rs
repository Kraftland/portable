#[derive(Debug)]
pub enum BindRule {
	/*
		Binds the file descriptor at path
		This SHOULD be preferred for user files and directories to avoid symlink attacks
		The descriptors path translates to --bind-fd, --ro-bind-fd
		(--dev-bind does not have a corresponding switch, so we need to reject these)

		WARNING: TYPE MUST NOT BE DEVICE
	*/
	FD {
		source_fd:	std::os::fd::OwnedFd,
		dest:		std::path::PathBuf,
		class:		BindType,
	},
	Path {
		source:		std::path::PathBuf,
		dest:		std::path::PathBuf,
		class:		BindType,
	},
	Symlink {
		source:		std::path::PathBuf,
		dest:		std::path::PathBuf,
	},
}

#[derive(Debug)]
pub enum BindType {
	ReadWrite,
	ReadOnly,
	Device,
}
