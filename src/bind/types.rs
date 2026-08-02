
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
				BindRule::Path { source, dest, class }	=> {
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
				BindRule::Tmpfs { dest }		=> {
					ret.push("--tmpfs".into());
					ret.push(dest.to_string_lossy().into());
				}
				BindRule::Symlink { source, dest }	=> {
					ret.push("--symlink".into());
					ret.push(source.to_string_lossy().into());
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
			}
		};
		Self {
			rules:	ret
		}
	}
}

