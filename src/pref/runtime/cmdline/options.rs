pub struct CmdlineOpts {
	/**
		File forwarding
	*/
	pub file_expose:	Option<std::collections::HashMap<std::path::PathBuf, std::path::PathBuf>>,
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
	Passthrough
}
