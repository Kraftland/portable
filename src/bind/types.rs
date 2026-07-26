
pub enum BindRule {
	// Binds the file descriptor at path
	// This SHOULD be preferred for user files and directories to avoid symlink attacks
	// The descriptors path translates to --bind-fd, --ro-bind-fd (--dev-bind does not have a corresponding switch, so we need to reject these)
	FD {
		source_fd:	std::os::fd::OwnedFd,
		dest:		std::path::PathBuf,
		read_only:	bool,
	},
	Symlink
}
