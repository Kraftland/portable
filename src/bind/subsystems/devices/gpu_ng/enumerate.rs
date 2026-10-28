/**
	GPUInfo represents a single GPU device.

	It contains information, for example if it should be bound when not driving the boot display.
*/
pub struct GPUInfo {
	/**
		boot_display describes whether the graphics card is used for booting.

		It may also be set to true if boot_vga is present on it's parent device.
	*/
	pub	boot_display:	bool,
}
