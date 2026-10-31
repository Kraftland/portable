/**
	A specific vendor.

	It implements the TryFrom trait to convert from Strings.
*/
#[derive(PartialEq, Eq)]
pub enum GPUVendor {
	Intel,
	AMD,
	NVIDIA {
		driver:	nvidia::NVIDIADriver,
	},
	Other {
		vendor:	String,
	},
}

pub mod nvidia;

/**
	The VendorHook trait is used to define vendor-specific actions.

	It has two options: present and absent.

	The present action is always called when the specific GPU is involved in exposing, but the
	absent action may only be called when the GPU is present, but not actively used.
*/
pub trait VendorHook {}
