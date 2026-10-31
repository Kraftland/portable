/**
	The public module convert implements TryFrom<super::GPUInfo> for Vec<crate::bind::types::BindRule>
	to convert from a GPU object to bind rules.
*/
pub mod convert;

mod select;
pub use select::select;

use super::GPUInfo;
