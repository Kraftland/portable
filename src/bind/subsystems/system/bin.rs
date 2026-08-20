#[derive(Debug, thiserror::Error)]
pub enum BinError {}

pub async fn bind(conf: std::sync::Arc<crate::config::Config>)
-> Result<crate::bind::types::BindRules, BinError> {
	use crate::bind::types::BindRule;

	let mut ret = vec![];

	let mut overlay_source = vec![
		std::path::PathBuf::from("/usr/bin"),
		std::path::PathBuf::from("/usr/lib/portable/overlay-usr"),
	];

	if conf.exec.overlay {
		let mut path = std::path::PathBuf::from("/usr/lib/portable/info");
		path.push(&conf.metadata.sandbox_id);
		path.push("bin");
		overlay_source.push(
			path
		);
	};

	ret.push(
		BindRule::Overlay {
			sources: overlay_source,
			dest: "/usr/bin".into(),
			class: crate::bind::types::OverlayType::Ro,
		}
	);

	Ok(ret)
}
