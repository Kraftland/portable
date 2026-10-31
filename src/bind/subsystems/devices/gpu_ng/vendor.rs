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

	For example, on a hybrid graphics laptop, Portable by default only binds the integrated GPU,
	commonly known as the iGPU, while leaving the discrete GPU behind. This is intended to save
	power, but it required special hacks to cover some kernel module paths.

	By introducing this trait and implementing this for GPUVendor, we can elegantly process
	workarounds for various vendors.
*/
pub trait VendorHook {
	/**
		Actions to invoke when absent, see trait document
	*/
	fn absent(&self) -> impl Future<Output = Result<crate::bind::types::BindRules, VendorError>>;
	/**
		Actions to invoke when present, see trait document
	*/
	fn present(&self) -> impl Future<Output = Result<crate::bind::types::BindRules, VendorError>>;
}

#[derive(Debug, thiserror::Error)]
pub enum VendorError {}
