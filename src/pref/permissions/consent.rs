/**
	The impls module hosts different backend for AskConsent and Permission Storage
*/
pub mod impls;

/**
	The public trait AskConsent is used to implement consent dialogue for different backends.

	The ConsentContent must be populated.

	ask() implements an async function for asking user consent, of which returns user agreed
		ConsentContent.
*/
pub trait AskConsent {
	fn ask(content: ConsentContent)
	-> impl std::future::Future<Output = Result<ConsentContent, Self::ConsentError>>;

	type ConsentError;
}

/**
	See UserConsent trait
*/
#[derive(PartialEq, Eq)]
pub struct ConsentContent {
	permissions:	Vec<portable_config::definitions::consent::DynamicPermission>,
}

impl ConsentContent {
	/**
		Merge two ConsentContent s
	*/
	pub fn merge(value1: Self, value2: Self) -> Self {
		let mut permissions = vec![];

		permissions.extend(value1.permissions);

		permissions.extend(value2.permissions);
		Self { permissions: permissions }
	}
}

impl From<Vec<portable_config::definitions::consent::DynamicPermission>> for ConsentContent {
	fn from(value: Vec<portable_config::definitions::consent::DynamicPermission>) -> Self {
		Self { permissions: value }
	}
}
