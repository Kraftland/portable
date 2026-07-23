use crate::config_definition::*;
use thiserror::Error;
use tokio::io::{AsyncReadExt};
//use std::fs::OpenOptions;

#[derive(Debug, Error)]
pub enum LegacyConfigError {
	#[error("Could not open legacy config: {0:#?}")]
	OpenConfigError(std::io::Error),

	#[error("Could not read legacy config: {0:#?}")]
	ReadConfigError(tokio::io::Error),

	#[error("Could not deserialise legacy config: {0:#?}")]
	DeserializeError(legacy_conf::Error),
}

pub async fn get_legacy_conf(path: &std::path::Path) -> Result<Config, LegacyConfigError> {
	let mut file = tokio::fs::OpenOptions::new()
		.read(true)
		.write(false)
		.create(false)
		.open(path)
		.await
		.map_err(LegacyConfigError::OpenConfigError)
		?;
	let mut config_raw = String::new();
	file
		.read_to_string(&mut config_raw)
		.await
		.map_err(LegacyConfigError::ReadConfigError)?;

	let decoded_legacy_conf: legacy_conf::Config
		= legacy_conf::from_str(config_raw.as_str())
			.map_err(LegacyConfigError::DeserializeError)
			?;

	let mut dev_allow = vec![];

	if decoded_legacy_conf.game {
		dev_allow.push(DeviceAllow::DiscreteGPU);
	};
	if decoded_legacy_conf.camera {
		dev_allow.push(DeviceAllow::Camera);
	};
	if decoded_legacy_conf.input_dev {
		dev_allow.push(DeviceAllow::Input);
	};

	Ok(Config {
		metadata: Metadata {
			sandbox_id: decoded_legacy_conf.app_id,
			display_name: decoded_legacy_conf.friendly_name,
			state_directory: decoded_legacy_conf.state_dir,
			config_version: 10,
		},
		exec: Exec {
			target: decoded_legacy_conf.target.0,
			arguments: decoded_legacy_conf.target.1.unwrap_or(vec![]),
			overlay: false,
		},
		dbus_activation: BusExec {
			enable: false,
			target: "".to_string(),
			arguments: vec![],
			overlay: false,
		},
		process: ProcMgmt {
			background: true,
		},
		system: SysMgmt {
			allow_inhibit: false,
			conduct_inhibit: false,
			uclamp_max: 100,
			device_allow: dev_allow,
		},
		network: Network {
			allow_network: decoded_legacy_conf.bind_network,
			enable_filter: false,
			block_dest: vec![],
		},
		privacy: Privacy {
			lockdown: false,
			x11_compat: ! decoded_legacy_conf.wayland,
			classic_notif: false,
			pipewire: false,
		},
		advanced: Advanced {
			use_zink: decoded_legacy_conf.zink,
			qt5_compat: decoded_legacy_conf.qt5,
			mpris_names: vec![],
			tray_wake: decoded_legacy_conf.tray_wake,
			allow_kde_status: false,
			flatpak_env: decoded_legacy_conf.flatpak_info,
			allow_debug: false,
		},
	})
}
