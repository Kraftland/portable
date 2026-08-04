#[derive(Debug, thiserror::Error)]
pub enum DisplayBindError {}

pub struct X11;

impl super::BindDisplay for X11 {
	async fn bind(self) -> Result<crate::bind::types::BindRules, Self::DisplayBindError> {
		unimplemented!()
	}

	type DisplayBindError = DisplayBindError;
}
