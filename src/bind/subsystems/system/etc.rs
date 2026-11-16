/**
	The public function bind_etc is used to generate bind mount rules for paths under /etc

	Only parts essential to application operation and theming under /etc is bound, such that
		attack and leak surface is down to a minimum.
*/
pub async fn bind_etc(
	config:			std::sync::Arc<crate::config::Config>,
	xdg:			std::sync::Arc<crate::xdg::XdgDirs>,
	portable_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
) -> Result<crate::bind::types::BindRules, EtcError> {
	let state_directory = {
		let mut path = xdg.data_home.to_path_buf();
		path.push(&config.metadata.state_directory);
		path
	};

	use crate::bind::types::BindRule;
	use crate::bind::subsystems::dirs::RuntimePathsTrait;

	let resolv_spawn = tokio::spawn(
		super::resolv::mount()
	);

	let passwd_spawn = tokio::spawn(
		super::passwd::generate(
			portable_runtime.path(),
			state_directory,
		)
	);
	let nsswitch_spawn = tokio::spawn(
		super::nsswitch::generate(portable_runtime.path())
	);

	let machine_id = tokio::spawn(
		super::machine_id::bind(
			config.clone(),
			xdg.clone(),
		),
	);

	let mut ret = vec![
		BindRule::Path {
			source:	{
				nsswitch_spawn
					.await
					.map_err(EtcError::SpawnError)?
					.map_err(EtcError::NsswitchError)?
			},
			dest:	"/etc/nsswitch.conf".into(),
			class:	crate::bind::types::BindType::ReadOnly,
		},
	];

	ret.extend(
		passwd_spawn
			.await
			.map_err(EtcError::SpawnError)?
			.map_err(EtcError::PasswdError)?
	);

	ret.extend(
		resolv_spawn
			.await
			.map_err(EtcError::SpawnError)?
			.map_err(EtcError::ResolvError)?
	);

	ret.extend(
		machine_id
			.await
			.map_err(EtcError::SpawnError)
			?
			.map_err(EtcError::MachineIDError)
			?
	);

	Ok(ret)
}

#[derive(Debug, thiserror::Error)]
pub enum EtcError {
	#[error("Could not spawn /etc bind task: {0}")]
	SpawnError(tokio::task::JoinError),

	#[error("Could not generate nsswitch file: {0}")]
	NsswitchError(super::nsswitch::NsswitchError),

	#[error("Could not generate passwd file: {0}")]
	PasswdError(super::passwd::PasswdError),

	#[error("Could not generate Machine ID: {0}")]
	MachineIDError(super::machine_id::MachineIDError),

	#[error("Could not bind resolv.conf: {0}")]
	ResolvError(super::resolv::ResolvError),
}
