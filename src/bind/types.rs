
/**
	BindRules represents a list of bind rules that is specifically without dependency tree.
	It is not meant to be read or manipulated by outside modules to ensure consistency.

	The trait ToCmdline is implemented to convert from BindRules to bubblewrap arguments.
*/
pub struct BindRules {
	rules:	Vec<BindRule>
}

/**
	BindRule represents a single rule of exposing the host system
*/
#[derive(Debug)]
pub enum BindRule {
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

	/**
		The sources are overlaid in the order given,
			with the first source on the command line at the bottom of the stack:
		if a  given path to be read exists in more than one source,
		the file is read from the last such source specified.
	*/
	Overlay {
		sources:	Vec<std::path::PathBuf>,
		dest:		std::path::PathBuf,
		class:		OverlayType,
	},
}

/**
	The type of overlayfs
*/
#[derive(Debug)]
pub enum OverlayType {
	/**
		With ReadWrite all writes will go to RWSRC.
		Reads will come preferentially from RWSRC,
		then from any --overlay-src paths.
		WORKDIR must be an empty directory on the same filesystem as RWSRC,
		and is used internally by the kernel.
	*/
	ReadWrite {
		rwsrc:		std::path::PathBuf,
		workdir:	std::path::PathBuf,
	},

	/**
		All writes will go to the tmpfs that hosts the sandbox root
	*/
	Tmpfs,

	/**
		Filesystem will be mounted read-only
	*/
	Ro,
}

/**
	Specifies the Bind Type for filesystem

	The device type is not implemented for overlayfs mounting
*/
#[derive(Debug)]
pub enum BindType {
	ReadWrite,
	ReadOnly,
	Device,
}

pub trait DeDupRules {
	fn dedup(self)	-> Self;
}


/**
	The trait ToCmdline defines shared behaviour to convert certain rules as command line
	arguments.

	For example, BindRules implements this to
*/
pub trait ToCmdline {
	async fn to_cmdline(self)	-> Vec<String>;
}

impl ToCmdline for BindRules {
	async fn to_cmdline(self)	-> Vec<String> {
		let mut ret = vec![];
		for rule in self.rules {
			match rule {
				BindRule::Path { source, dest, class }		=> {
					match class {
						BindType::Device	=> {
							ret.push("--dev-bind".to_string());
						}
						BindType::ReadOnly	=> {
							ret.push("--ro-bind".into());
						}
						BindType::ReadWrite	=> {
							ret.push("--bind".into());
						}
					};
					ret.push(source.to_string_lossy().into());
					ret.push(dest.to_string_lossy().into());
				}
				BindRule::Tmpfs { dest }			=> {
					ret.push("--tmpfs".into());
					ret.push(dest.to_string_lossy().into());
				}
				BindRule::Symlink { source, dest }		=> {
					ret.push("--symlink".into());
					ret.push(source.to_string_lossy().into());
					ret.push(dest.to_string_lossy().into());
				}
				BindRule::Overlay { sources, dest, class }	=> {
					for source in sources {
						ret.push("--overlay-src".into());
						ret.push(source.to_string_lossy().into());
					};
					match class {
						OverlayType::Ro		=> {
							ret.push("--ro-overlay".into());
						}
						OverlayType::Tmpfs	=> {
							ret.push("--tmp-overlay".into());
						}
						OverlayType::ReadWrite { rwsrc, workdir }
									=> {
							ret.push("--overlay".into());
							ret.push(rwsrc.to_string_lossy().into());
							ret.push(workdir.to_string_lossy().into());
						}
					};
					ret.push(dest.to_string_lossy().into());
				}
			}
		};
		ret
	}
}


impl DeDupRules for BindRules {
	fn dedup(self)	-> Self {
		let mut ret = vec![];
		let mut dest_mnt = vec![];

		for rule in self.rules {
			match rule {
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
				BindRule::Overlay { sources, dest, class }
									=> {
					if dest_mnt.contains(&dest) {
						continue;
					};
					ret.push(BindRule::Overlay { sources, dest, class });
				}
			}
		};
		Self {
			rules:	ret
		}
	}
}

