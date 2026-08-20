mod bw_bind_par;
mod args;
pub mod share_file;
mod permission_reset;

pub use permission_reset::{reset, ResetError};

/**
	Parse the command line collection, it also handles some legacy environment variables.

	Actions are not performed here, instead they are handled in main thread.
*/
pub async fn parse(logger: crate::logger::LogSender) -> Result<super::options::RuntimeOpts, RuntimeOptsError> {
	let mut options = args::parse(
		logger.clone(),
	)
		.await
		.map_err(RuntimeOptsError::CmdlineError)
		?;

	match bw_bind_par::get_bwbindpar_opts() {
		Some(v)	=> {
			options.file_expose.push(v);
		}
		None	=> {}
	};

	Ok(options)
}

#[derive(thiserror::Error, Debug)]
pub enum RuntimeOptsError {
	#[error("Error parsing command line arguments: {0:#?}")]
	CmdlineError(args::CmdlineError)
}
