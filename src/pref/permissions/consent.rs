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
		config:		std::sync::Arc<crate::config::Config>,
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
	fn store(&self, perms: &DynamicPermissionsResult, app_id: std::sync::Arc<&str>) -> impl std::future::Future<Output = Result<(), Self::StoreError>>;

	/**
		Retrieve a list of DynamicPermissions from a PermissionStore backend
	*/
	fn retrieve(&self, app_id: std::sync::Arc<&str>) -> impl std::future::Future<Output = Result<DynamicPermissionsResult, Self::StoreError>>;

	type StoreError;
}
