/**
	The vendor module hosts vendor specific fixes.

	Stuff like fixing NVIDIA wake ups should go there.
*/
mod vendor;

/**
	The enumerate module hosts GPU listing code.

	It populates the GPUInfo struct at once to reduce multiple lookups to the udev subsystem.

	This design would also allow non-systemd (non-thread-safe) udev implementations to be used.
*/
mod enumerate;
pub use enumerate::GPUInfo;

/**
	The bind module decide which GPU to bind.

	It also implements the TryFrom<GPUBind> trait for Vec<BindRule>, of which is used to output
	the final rules.
*/
mod bind;
