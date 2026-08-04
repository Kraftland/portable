#[derive(Debug, thiserror::Error)]
pub enum DisplayBindError {}

pub struct Wayland;

impl super::BindDisplay for Wayland {
	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::DisplayBindError> {
		unimplemented!()
	}

	type DisplayBindError = DisplayBindError;
}
