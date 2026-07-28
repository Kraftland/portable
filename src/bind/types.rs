
/*
	BindRules represents a list of bind rules that is specifically without dependency tree.
	It is not meant to be read or manipulated by outside modules to ensure consistency
*/
pub struct BindRules {
	rules:	Vec<BindRule>
}

pub trait DeDupRules {
	fn dedup(self)	-> Self;
}


impl DeDupRules for BindRules {
	fn dedup(self)	-> Self {
		let mut ret = vec![];
		let mut dest_mnt = vec![];

		for rule in self.rules {
			match rule {
				BindRule::FD { source_fd, dest, class }	=> {
					if dest_mnt.contains(&dest) {
						continue;
					} else {
						dest_mnt.push(dest.clone());
						ret.push(BindRule::FD { source_fd, dest, class });
					};
				}
				BindRule::Path { source, dest, class }	=> {
					if dest_mnt.contains(&dest) {
						continue;
					} else {
						dest_mnt.push(dest.clone());
						ret.push(BindRule::Path { source, dest, class });
					};
				}
				BindRule::Symlink { source, dest }	=> {
					if dest_mnt.contains(&dest) {
						continue;
					} else {
						dest_mnt.push(dest.clone());
						ret.push(BindRule::Symlink { source, dest });
					};
				}
				BindRule::Tmpfs { dest }		=> {
					ret.push(BindRule::Tmpfs { dest });
				}
			}
		};
		Self {
			rules:	ret
		}
	}
}


/*
	BindRule represents a single rule of exposing the host system
*/
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
	Tmpfs {
		dest:		std::path::PathBuf,
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
