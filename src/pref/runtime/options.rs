pub struct RuntimeOpts {
	/**
		File forwarding
	*/
	pub file_expose:	Vec<FileExposurePreference>,
	pub Action:		Action,
	pub app_args:		Vec<String>,
}

/**
	StartMode designates different operation modes

	Normal means continue starting or activating the instance

	Action designates the previous --actions flag

	When set to other values, the main thread aborts start up
	and said actions should be handled in the cmdline module
*/
pub enum Action {
	Normal {
		debug_shell:	bool,
	},
	ShareFile,
	ShareDir,
	OpenHome,
	ResetDocs,
	ShowStats,
	Quit,
}

/**
	This enum represents multiple possible ways for exposing a file

	It is not the final representation of Files Map, as processing is needed to handle some files
*/
pub enum FileExposurePreference {
	/**
		Essentially bubblewrap's --bind flag, and BindRule::Path

		Does not work on secondary instances since we can't really bind-mount after chroot

		Cmdline rewrite is needed for this type
	*/
	MountPath {
		host:	std::path::PathBuf,
		dest:	std::path::PathBuf,
		class:	crate::bind::types::BindType,
	},

	/**
		Pass the file inside sandbox using XDG Desktop Portals

		Works on secondary instances

		Cmdline rewrite is needed for this type
	*/
	Passthrough {
		host:	std::path::PathBuf,
	},
}
