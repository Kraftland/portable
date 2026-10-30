/**
	GPUInfo represents a single GPU device.

	It contains information, for example if it should be bound when not driving the boot display.
*/
#[derive(PartialEq, Eq)]
pub struct GPUInfo {
	/**
		boot_display describes whether the graphics card is used for booting.

		It may also be set to true if boot_vga is present on it's parent device.
	*/
	pub	boot_display:	bool,

	pub	vendor:		super::vendor::GPUVendor,

	/**
		Renderer node name. Like renderD128, renderD129.

		Obtained via sysname()
	*/
	pub	render_name:	String,

	/**
		devlinks for the renderer node
	*/
	pub	render_devlink:	Vec<std::path::PathBuf>,


	/**
		Kernel device node for the renderer
	*/
	pub	render_devnode:	Option<std::path::PathBuf>,

	/**
		Device path in sysfs
	*/
	pub	render_syspath:	Option<std::path::PathBuf>,

	/**
		Card node name. Like card0, card1.

		Obtained via sysname()
	*/
	pub	card_name:	String,
	/**
		devlinks for the card node
	*/
	pub	card_devlink:	Vec<std::path::PathBuf>,
	/**
		Kernel device node for the card node
	*/
	pub	card_devnode:	Option<std::path::PathBuf>,
	/**
		Device path in sysfs
	*/
	pub	card_syspath:	Option<std::path::PathBuf>,

	/**
		The parent device is included because some applications need them, like Lutris

		devlinks for the parent device
	*/
	pub	parent_devlink:	Vec<std::path::PathBuf>,
	/**
		Kernel device node for the parent device
	*/
	pub	parent_devnode:	Option<std::path::PathBuf>,
	/**
		Device path in sysfs
	*/
	pub	parent_syspath:	Option<std::path::PathBuf>,
}
