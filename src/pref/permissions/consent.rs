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

pub trait AskConsent {
	fn ask(content: DynamicPermissions)
	-> impl std::future::Future<Output = Result<DynamicPermissions, Self::ConsentError>>;

	type ConsentError;
}
