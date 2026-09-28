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

	let mut stored_permissions = match portal_store.retrieve(&config.metadata.sandbox_id).await {
		Ok(v)	=> {
			#[cfg(debug_assertions)]
			let _ = logger.send(
				crate::logger::LogMessage {
					level:		crate::logger::LogLevel::Warn,
					message:	format!(
						"Retrieved permissions from backend: {v:#?}",
					),
				},
			).await;

			v
		},
		Err(e)	=> {
			let _ = logger.send(
				crate::logger::LogMessage {
					level:		crate::logger::LogLevel::Warn,
					message:	format!("Could not retrieve stored permissions: {e}"),
				}
			).await;
			std::collections::HashMap::new()
		}
	};

	let config_permissions: DynamicPermissions = config.as_ref().into();

	let unknown_permissions = {
		let mut ret = vec![];

		for perm in config_permissions {
			if stored_permissions.contains_key(&perm)
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

	stored_permissions.extend(
		DynamicPermissions::ask(&config, unknown_permissions)
			.await
			.map_err(ConsentError::ZenityError)
			?
	);

	portal_store.store(&stored_permissions, &config.metadata.sandbox_id)
		.await
		.map_err(ConsentError::StorePortalError)
		?;

	Ok(
		stored_permissions.into()
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
pub type DynamicPermissionsResult = std::collections::HashMap<portable_config::definitions::consent::DynamicPermission, bool>;

/**
	The AskConsent trait is implemented by various backends to present user with a dialogue.

	It takes a list of DynamicPermission, and return them in a hashed favour.
*/
pub trait AskConsent {
	fn ask(
		config:		&crate::config::Config,
		missing_perms:	DynamicPermissions,
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
