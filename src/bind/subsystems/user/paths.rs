
pub async fn bind(
	xdg_data_dir:	std::path::PathBuf,
	state_dir:	&str,
) -> Result<crate::bind::types::BindRules, super::UserBindError> {
	use crate::bind::types::BindRule;

	let sandbox_home = {
		let mut home = xdg_data_dir.to_path_buf();
		home.push(&state_dir);
		home
	};

	super::create_state_dir::create_state_dir(&sandbox_home).await?;

	let ret = vec![
		BindRule::Path {
			source: sandbox_home.to_path_buf(),
			dest: sandbox_home.to_path_buf(),
			class: crate::bind::types::BindType::ReadWrite,
		},
	];

	Ok(ret)
}
