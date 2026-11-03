/**
	The vendor module hosts vendor specific fixes.

	Stuff like fixing NVIDIA wake ups should go there.
*/
mod vendor;

pub use vendor::VendorError;

/**
	The enumerate module hosts GPU listing code.

	It populates the GPUInfo struct at once to reduce multiple lookups to the udev subsystem.

	This design would also allow non-systemd (non-thread-safe) udev implementations to be used.
*/
mod enumerate;
pub use enumerate::GPUInfo;
pub use enumerate::EnumerateError;

/**
	The bind module decide which GPU to bind.

	It also implements the TryFrom<GPUBind> trait for Vec<BindRule>, of which is used to output
	the final rules.
*/
mod bind;
pub use bind::select::SelectError;

pub async fn bind(
	all_gpus:	bool,
	zink:		bool,
	logger:		crate::logger::LogSender,
	env:		crate::envs::holder::HoldChannel,
) -> Result<crate::bind::types::BindRules, GPUError> {
	let gpus = match enumerate::from::enumerate().await {
		Ok(v)	=> v,
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::messages::LogMessage::Warn(
					crate::logger::messages::warn::WarnMessage::GPUEnumerateError(e)
				)
			).await;

			return Ok(vec![]);
		}
	};

	Ok(
		crate::bind::types::DeDupRules::dedup(
			bind::select::select(
				gpus,
				&all_gpus,
				&zink,
				logger,
				env,
			)
				.await
		)
	)
}

#[derive(Debug, thiserror::Error)]
pub enum GPUError {}
