/**
	The get() function outputs a list of allowed consents as DynamicPermissions (type alias).

	It implements the core comparing logic, does query on backends and present a dialogue.
*/
pub async fn get(
	logger:	crate::logger::LogSender,
	config:	std::sync::Arc<portable_config::Config>,
	bus:	zbus::Connection,
) -> Result<std::sync::Arc<DynamicPermissionsResult>, ConsentError> {
	let portal_store = impls::store::portal::Portal {
		bus:	bus,
		logger:	logger.clone(),
	};

	let stored_permissions = match portal_store.retrieve(&config.metadata.sandbox_id).await {
		Ok(v)	=> v,
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level:		crate::logger::LogLevel::Warn,
					message:	format!("Could not retrieve stored permissions: {e}"),
				}
			).await;
			vec![]
		}
	};

	let config_permissions: DynamicPermissions = config.as_ref().into();

	let unknown_permissions = {
		let mut ret = vec![];

		for perm in config_permissions {
			if stored_permissions
				.iter()
				.any(
					|(perms, _)| {
						perms == &perm
					}
				)
			{
				continue;
			} else {
				ret.push(perm);
			}
		};

		ret
	};

	if unknown_permissions.len() == 0 {
		return Ok(
			stored_permissions.into()
		);
	};

	let ask_result = {
		DynamicPermissions::ask(unknown_permissions, config.as_ref())
			.await
			.map_err(ConsentError::ZenityError)
			?
	};

	let merged_result = {
		let mut ret = vec![];

		ret.extend(ask_result);

		ret.extend(stored_permissions);

		ret
	};

	portal_store.store(&merged_result, &config.metadata.sandbox_id)
		.await
		.map_err(ConsentError::StorePortalError)
		?;

	Ok(
		merged_result.into()
	)
}

#[derive(thiserror::Error, Debug)]
pub enum ConsentError {
	#[error("Could not retrieve stored permissions: {0:?}")]
	RetrievePortalError(impls::store::portal::PortalPermissionStoreError),

	#[error("Could not store permissions: {0:?}")]
	StorePortalError(impls::store::portal::PortalPermissionStoreError),

	#[error("Could not query user consent via Zenity: {0}")]
	ZenityError(impls::ask::zenity::ZenityError),
}

/**
	The impls module hosts different backend for AskConsent and Permission Storage
*/
pub mod impls;

/**
	The public trait AskConsent is used to implement consent dialogue for different backends.

	It is implemented for the DynamicPermissions struct by modules under consent/impls/ask/
		to delegate permission dialogue.

	The DynamicPermissions must be populated.

	ask() implements an async function for asking user consent, of which returns user agreed
		DynamicPermissions.
*/

pub type DynamicPermissions = Vec<portable_config::definitions::consent::DynamicPermission>;
pub type DynamicPermissionsResult = Vec<(portable_config::definitions::consent::DynamicPermission, bool)>;

pub trait AskConsent {
	fn ask(
		content:	DynamicPermissions,
		config:		&crate::config::Config,
	)
	-> impl std::future::Future<Output = Result<DynamicPermissionsResult, Self::ConsentError>>;

	type ConsentError;
}

/**
	See impls/store/portal.rs for detailed information on how the permission is stored for now
*/
pub trait PermissionStore {
	/**
		Store a list of Dynamic Permissions to a PermissionStore backend
	*/
	fn store(&self, perms: &DynamicPermissionsResult, app_id: &str) -> impl std::future::Future<Output = Result<(), Self::StoreError>>;

	/**
		Retrieve a list of DynamicPermissions from a PermissionStore backend

		Implementations should return an empty vector if not initialised.
	*/
	fn retrieve(&self, app_id: &str) -> impl std::future::Future<Output = Result<DynamicPermissionsResult, Self::StoreError>>;

	type StoreError;
}
