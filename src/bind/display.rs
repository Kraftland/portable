#[derive(thiserror::Error, Debug)]
pub enum DisplayError {}

use crate::bind::types::BindRules;

pub async fn bind() -> Result<BindRules, DisplayError> {
	unimplemented!();
	vec![]
}
