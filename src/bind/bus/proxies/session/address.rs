/**
	Get the socket address for proxy socket, and generate host sandbox rules
*/
pub async fn get_address_with_sandbox(
	portable_dir:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
)
-> Result<(std::path::PathBuf, crate::bind::types::BindRules), super::ProxyError> {

		let proxy_parent_path = {
			use crate::bind::subsystems::dirs::RuntimePathsTrait;
			let mut path = portable_dir.path();
			path.push("session");
			tokio::fs::create_dir_all(&path)
				.await
				.map_err(super::ProxyError::IOError)
				?;

			println!("Created bus proxy parent: {0:?}", path);

			path
		};

		let proxy_socket = {
			let mut path = proxy_parent_path.to_path_buf();
			path.push("bus");
			path
		};

		let host_sandbox = vec![
			crate::bind::types::BindRule::Path {
				source: proxy_parent_path.to_path_buf(),
				dest: "/run/session_bus".into(),
				class: crate::bind::types::BindType::ReadOnly,
			},
			crate::bind::types::BindRule::Symlink {
				source:	"/run/session_bus/bus".into(),
				dest:	"/run/sessionBus".into(),
			}
		];



		Ok((proxy_socket, host_sandbox))
}
