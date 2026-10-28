/**
	A specific vendor.

	It implements the TryFrom trait to convert from Strings.
*/
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
