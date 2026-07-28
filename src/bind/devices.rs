use thiserror::Error;

mod GPU;

#[derive(Debug, Error)]
pub enum EnumerateError {
	#[error("Could not enumerate devices: create enumerator failed: {0:#?}")]
	CreateEnumeratorError(String),
	#[error("Could not enumerate devices: add match failed: {0:#?}")]
	AddMatchError(std::io::Error),
	#[error("Could not enumerate devices: scan failed: {0:#?}")]
	ScanError(std::io::Error),
}

#[derive(Debug)]
pub enum Filter {
	// Enumerate by subsystem, this implies initialised
	Subsystem {subsystem: String},
}

use udev::{Device};
pub async fn enumerate(filter: Filter) -> Result<Vec<Device>, EnumerateError> {
	let mut enumerator = match filter {
		Filter::Subsystem { subsystem }	=> {
			let mut enumerator = {
				match udev::Enumerator::new() {
					Ok(v)	=> {v}
					Err(e)	=> {
						return Err(
							EnumerateError::CreateEnumeratorError(
								format!("{e:#?}"),
							),
						);
					}
				}
			};
			enumerator
				.match_is_initialized()
				.map_err(EnumerateError::AddMatchError)
				?;
			enumerator
				.match_subsystem(subsystem)
				.map_err(EnumerateError::AddMatchError)
				?;

			enumerator
		}
	};
	let list = enumerator
		.scan_devices()
		.map_err(EnumerateError::ScanError)?;
	let mut ret = vec![];
	for dev in list {
		ret.push(dev);
	};
	Ok(ret)
}
