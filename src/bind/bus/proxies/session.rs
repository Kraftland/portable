/*
	Proxy for the D-Bus session bus address
*/

impl crate::bind::bus::StartProxy for crate::bind::bus::Proxy {
	async fn start(
			proxy: crate::bind::bus::Proxy,
		) -> Result<(), Self::ProxyError>
	{

	}

	async fn new(
			logger:		crate::logger::LogSender,
			proxy_path:	std::path::PathBuf,

			#[cfg(feature = "flatpak")]
			info_path:	std::path::PathBuf,
			#[cfg(feature = "flatpak")]
			runtime_dir:	std::path::PathBuf,
		) -> Result<Self, Self::ProxyError>
	{
		compile_rules(
			logger,
			proxy_path,

			#[cfg(feature = "flatpak")]
			info_path,
			#[cfg(feature = "flatpak")]
			runtime_dir
		).await
	}

	type ProxyError = ProxyError;

}


use crate::bind::bus::Proxy;
use crate::bind::types::BindRule;
async fn compile_rules(
	logger:		crate::logger::LogSender,
	proxy_path:	std::path::PathBuf,

	#[cfg(feature = "flatpak")]
	info_path:	std::path::PathBuf,
	#[cfg(feature = "flatpak")]
	runtime_dir:	std::path::PathBuf,
) -> Result<Proxy, ProxyError> {
	let bus_address = tokio::spawn(get_session_bus_address());

	let proxy_address = {
		let addr = proxy_path.as_os_str().to_string_lossy();
		let mut path = String::from("unix:path=");
		path.push_str(&addr);
		path
	};

	let sandbox_rules = tokio::spawn(
		generate_sandbox_rules(
			proxy_path.clone(),
			#[cfg(feature = "flatpak")]
			info_path,
			#[cfg(feature = "flatpak")]
			runtime_dir,
		),
	);


	let bus_access: Vec<crate::bind::bus::rules::BusAccessLevel> = vec![];

	Ok(
		Proxy {
			sandbox: sandbox_rules
				.await
				.map_err(ProxyError::SpawnError)
				?
				?,
			bus_access: bus_access,
			bus_address: bus_address
				.await
				.map_err(ProxyError::SpawnError)
				?
				?,
			logger: logger,
			proxy_address: proxy_address,
		}
	)
}

async fn generate_bus_rules(
	app_id:	&str,
) -> Result<Vec<crate::bind::bus::rules::BusAccessLevel>, ProxyError> {
	use crate::bind::bus::rules::BusAccessLevel;
	use crate::bind::bus::rules::BusName;
	let mut rules: Vec<BusAccessLevel> = vec![
		BusAccessLevel::OwnName {
			bus_name: {
				let mut name = String::from(app_id);
				name.push_str(".*");
				BusName::try_from(name)
					.map_err(ProxyError::InvalidBusNameError)
					?
			},
		},
		BusAccessLevel::OwnName {
			bus_name: {
				let name = String::from(app_id);
				BusName::try_from(name)
					.map_err(ProxyError::InvalidBusNameError)
					?
			},
		},
		BusAccessLevel::WellknownName {
			bus_name: BusName::try_from("org.unifiedpush.Distributor.*")
				.map_err(ProxyError::InvalidBusNameError)
				?,
		}
	];




	Ok(rules)
}

async fn get_session_bus_address() -> Result<String, ProxyError> {
	let env = std::env::var("DBUS_SESSION_BUS_ADDRESS")
		.map_err(ProxyError::AddressUnknownError)
		?;
	Ok(env)
}

async fn generate_sandbox_rules(
	proxy_path:	std::path::PathBuf,

	#[cfg(feature = "flatpak")]
	info_path:	std::path::PathBuf,
	#[cfg(feature = "flatpak")]
	runtime_dir:	std::path::PathBuf,
) -> Result<crate::bind::types::BindRules, ProxyError> {
	let mut rules = vec![
		BindRule::Symlink {
			source: "/usr/lib64".into(),
			dest: "/lib64".into(),
		},
		BindRule::Path {
			source: "/usr/lib".into(),
			dest: "/usr/lib".into(),
			class: crate::bind::types::BindType::ReadOnly,
		},
		BindRule::Path {
			source: "/usr/lib64".into(),
			dest: "/usr/lib64".into(),
			class: crate::bind::types::BindType::ReadOnly,
		},
		BindRule::Path {
			source: "/usr/bin".into(),
			dest: "/usr/bin".into(),
			class: crate::bind::types::BindType::ReadOnly,
		},
		// BindRule::Path {
		// 	source: "/usr/share".into(),
		// 	dest: "/usr/share".into(),
		// 	class: crate::bind::types::BindType::ReadOnly,
		// },
		BindRule::Path {
			source: "/usr/bin".into(),
			dest: "/usr/bin".into(),
			class: crate::bind::types::BindType::ReadOnly,
		},

		BindRule::Path {
			source: proxy_path.clone(),
			dest: proxy_path,
			class: crate::bind::types::BindType::ReadWrite,
		},
	];

	#[cfg(feature = "flatpak")]
	{
		rules.push(
			BindRule::Path {
				source: info_path.clone(),
				dest: {
					let mut path = std::path::PathBuf::from(runtime_dir);
					path.push(".flatpak-info");
					path
				},
				class: crate::bind::types::BindType::ReadOnly,
			}
		);
		rules.push(
			BindRule::Path {
				source: info_path,
				dest: std::path::PathBuf::from("/.flatpak-info"),
				class: crate::bind::types::BindType::ReadOnly,
			}
		);
	}

	{
		let env = std::env::var("DBUS_SESSION_BUS_ADDRESS")
			.map_err(ProxyError::AddressUnknownError)
			?;
		let path = env.strip_prefix("unix:path=");
		match path {
			Some(v)	=> {
				rules.push(
					BindRule::Path {
						source: std::path::PathBuf::from(v),
						dest: std::path::PathBuf::from(v),
						class: crate::bind::types::BindType::ReadWrite,
					}
				);
			}
			None	=> {}
		}
	};

	Ok(rules)
}

use thiserror::Error;
#[derive(Debug, Error)]
pub enum ProxyError {
	#[error("Could not start D-Bus proxy for session bus: invalid address: {0:#?}")]
	AddressUnknownError(std::env::VarError),

	#[error("Could not start D-Bus proxy for session bus: thread spawn error: {0:#?}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not start D-Bus proxy for session bus: invalid bus name: {0:#?}")]
	InvalidBusNameError(crate::bind::bus::rules::BusNameError),

	#[error("Could not start D-Bus proxy for session bus: invalid character: {0:#?}")]
	OsStringError(std::io::Error),
}
