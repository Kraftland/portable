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
		Card node name. Like card0, card1.

		Obtained via sysname()
	*/
	pub	card_name:	String,
}
