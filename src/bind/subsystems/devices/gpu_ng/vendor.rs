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

mod from;

pub mod nvidia;
pub mod amd;

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

	Note that absent must run before present to make sure it does not override each other.
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

	/**
		Environment variables to set when present and all_gpus are set to true.

		Effectly just PRIME render offloading.
	*/
	fn offload(&self, env: &crate::envs::holder::HoldChannel)
		-> impl Future<Output = Result<(), VendorError>>;
}

impl VendorHook for GPUVendor {
	async fn absent(&self) -> Result<crate::bind::types::BindRules, VendorError> {
		match self {
			Self::Intel	=> {
				Ok(vec![])
			}
			Self::AMD	=> {
				Ok(vec![])
			}
			Self::NVIDIA { driver }
			=> {
				match driver {
					nvidia::NVIDIADriver::Proprietary	=> {
						nvidia::nvidia_module_mounts(true)
							.await
					}
					nvidia::NVIDIADriver::Unknown		=> {
						Err(
							VendorError::UnknownNVIDIADriver
						)
					}
					_					=> {
						Ok(vec![])
					}
				}
			}
			Self::Other { vendor }
			=> {
				Err(VendorError::UnknownVendor(vendor.to_string()))
			}
		}
	}

	async fn present(&self) -> Result<crate::bind::types::BindRules, VendorError> {
		match self {
			Self::Intel	=> {
				Ok(vec![])
			}
			Self::AMD	=> {
				amd::bind_rocm_device()
					.await
			}
			Self::NVIDIA { driver }
			=> {
				match driver {
					nvidia::NVIDIADriver::Proprietary	=> {
						let mut ret = nvidia::nvidia_module_mounts(false)
							.await
							?;

						for path in nvidia::get_nvidia_devices()
							.await
							?
						{
							ret.push(
								crate::bind::types::BindRule::Path {
									source:	path.to_path_buf(),
									dest:	path,
									class:	crate::bind::types::BindType::Device,
								}
							);
						};
						Ok(ret)
					}
					nvidia::NVIDIADriver::Unknown		=> {
						Err(
							VendorError::UnknownNVIDIADriver
						)
					}
					_					=> {
						Ok(vec![])
					}
				}
			}
			Self::Other { vendor }
			=> {
				Err(VendorError::UnknownVendor(vendor.to_string()))
			}
		}
	}

	async fn offload(&self, env: &crate::envs::holder::HoldChannel)
			-> Result<(), VendorError>
	{
		match self {
			Self::Intel | Self::AMD | Self::Other { vendor: _ }
			=> {
				// TODO: we should set device / vendor ID, see mesa:
				// https://docs.mesa3d.org/envvars.html#:~:text=DRI%5FPRIME
				// but we should also wait until gpud or some other selection logic
				// comes along, or we should probably just leave it as is.

				env.send(
					crate::envs::holder::EnvMessage::Add {
						key:	"DRI_PRIME".into(),
						value:	"1".into(),
					}
				)
					.await
					.map_err(VendorError::EnvSendError)
					?;
				Ok(())
			}

			Self::NVIDIA { driver }
			=> {
				match driver {
					nvidia::NVIDIADriver::Proprietary	=> {
						env.send(
							crate::envs::holder::EnvMessage::Add {
								key:	"__NV_PRIME_RENDER_OFFLOAD".into(),
								value:	"1".into(),
							}
						)
							.await
							.map_err(VendorError::EnvSendError)
							?;
						env.send(
							crate::envs::holder::EnvMessage::Add {
								key:	"__VK_LAYER_NV_optimus".into(),
								value:	"NVIDIA_only".into(),
							}
						)
							.await
							.map_err(VendorError::EnvSendError)
							?;
						env.send(
							crate::envs::holder::EnvMessage::Add {
								key:	"__GLX_VENDOR_LIBRARY_NAME".into(),
								value:	"nvidia".into(),
							}
						)
							.await
							.map_err(VendorError::EnvSendError)
							?;
						Ok(())
					}
					nvidia::NVIDIADriver::Nouveau | nvidia::NVIDIADriver::Unknown
					=> {
						env.send(
							crate::envs::holder::EnvMessage::Add {
								key:	"DRI_PRIME".into(),
								value:	"1".into(),
							}
						)
							.await
							.map_err(VendorError::EnvSendError)
							?;
						Ok(())
					}
				}
			}
		}
	}
}

#[derive(Debug, thiserror::Error)]
pub enum VendorError {
	#[error("Could not send PRIME render offload variables: {0}")]
	EnvSendError(tokio::sync::mpsc::error::SendError<crate::envs::holder::EnvMessage>),

	#[error("Could not scan for nvidia devices under devtmpfs: {0}")]
	ScanNvidiaDevError(std::io::Error),

	#[error("Could not mount or unmount nvidia modules: {0}")]
	MountNvidiaModuleError(std::io::Error),

	#[error("Could not mount ROCm device: {0}")]
	KfdIOError(std::io::Error),

	#[error("Unknown GPU vendor: {0}")]
	UnknownVendor(String),

	#[error("Unknown NVIDIA driver type")]
	UnknownNVIDIADriver,
}
