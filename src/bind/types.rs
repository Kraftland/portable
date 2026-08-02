
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

	/**
		Mount new virtual filesystems on DEST (devtmpfs, etc.)
	*/
	VirtualFS {
		dest:		std::path::PathBuf,
		class:		VirtualFS,
	}
}

/**
	The type of VFS
*/
#[derive(Debug)]
pub enum VirtualFS {
	/// Mount new devtmpfs
	Devtmpfs,
	/// Mount new procfs
	Procfs,
	/// Mount new tmpfs
	Tmpfs {
		/**
			Specify a size limit.
			The value is internally translated to bytes my multiplying 1024^2
			and passed to --size in bubblewrap
		*/
		size_mb:	Option<usize>,
		perms:		Option<std::fs::Permissions>,
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
				BindRule::VirtualFS { dest, class }		=> {
					match class {
						VirtualFS::Devtmpfs	=> {
							ret.push("--dev".into());
						}
						VirtualFS::Procfs	=> {
							ret.push("--proc".into());
						}
						VirtualFS::Tmpfs { size_mb, perms }
									=> {
							match size_mb {
								Some(v)	=> {
									ret.push("--size".into());
									let size = v * 1024 * 1024;
									ret.push(size.to_string());
								}
								None	=> {}
							}
							match perms {
								Some(v)	=> {
									use std::os::unix::fs::PermissionsExt;
									ret.push("--perms".into());
									ret.push(format!("{:04o}", v.mode()));
								}
								None	=> {}
							}
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
				BindRule::Overlay { sources, dest, class }
									=> {
					if dest_mnt.contains(&dest) {
						continue;
					};
					dest_mnt.push(dest.clone());
					ret.push(BindRule::Overlay { sources, dest, class });
				}
				BindRule::VirtualFS { dest, class }	=> {
					if dest_mnt.contains(&dest) {
						continue;
					};
					dest_mnt.push(dest.clone());
					ret.push(BindRule::VirtualFS { dest, class });
				}
			}
		};
		Self {
			rules:	ret
		}
	}
}

