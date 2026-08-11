/**
	The public struct InitInfo describes information passed down to Init via bus IPC

	It is estimated that passing them directly instead of using memfd is faster at smaller
	quantity.

	With the new way of passing down information, Init will be supplied of only an app_id, and
	will therefore contact the controller. Thus, we can manipulate the started atomic boolean
	inside AuxStart struct to clearly indicate whether the Init system has started.
*/
pub struct InitInfo {
	pub extra_files:	std::collections::HashMap<String, String>,
	pub inhibit_suspend:	bool,
	pub flatpak_info:	bool,

	/**
		Lockdown is an alias of seccomp whitelist + landlock
	*/
	pub lockdown:		bool,

	/**
		Whether or not to allow a set of debugging syscalls
	*/
	pub allow_debug:	bool,

	/**
		Designates the target executable to start upon.

		Care should be taken when constructing this field, because debug shell and
			D-Bus activation can have different target executable.
	*/
	pub target_exec:	String,

	/**
		An array of strings describing the arguments to pass
	*/
	pub target_args:	Vec<String>,

	/**
		uclamp_min describes the minimum guaranteed performance operating point.

		It is clamped between 0 and 100, as per cgroup v2 specifications.
	*/
	pub uclamp_min:		u8,
	/**
		uclamp_max describes the maximum performance operating point.
		It operates as a ceiling limit.

		It is clamped between 0 and 100, as per cgroup v2 specifications.
	*/
	pub uclamp_max:		u8,
}
