
pub async fn bind(
	xdg_data_dir:	std::sync::Arc<std::path::PathBuf>,
	xdg_home:	std::sync::Arc<std::path::PathBuf>,
	state_dir:	std::sync::Arc<str>,
) -> Result<crate::bind::types::BindRules, super::UserBindError> {
	use crate::bind::types::BindRule;

	let sandbox_home = {
		let mut home = xdg_data_dir.to_path_buf();
		home.push(&*state_dir);
		home
	};

	super::create_state_dir::create_state_dir(&sandbox_home).await?;

	let ret = vec![
		BindRule::Path {
			source: sandbox_home.clone(),
			dest: sandbox_home.clone(),
			class: crate::bind::types::BindType::ReadWrite,
		},
		/*
			We are using symlink here to avoid screwing up and exposing preferences
		*/
		BindRule::Symlink {
			source:	sandbox_home,
			dest:	xdg_home.to_path_buf(),
		},
	];

	Ok(ret)
}
