/**
	The public trait UserConsent is used to implement consent dialogue for different backends.

	The ConsentContent must be populated.
*/
pub trait UserConsent {
	fn ask(content: ConsentContent)
	-> impl std::future::Future<Output = Result<ConsentContent, Self::ConsentError>>;

	type ConsentError;
}

/**
	See UserConsent trait
*/
pub struct ConsentContent {
	permissions:	Vec<portable_config::definitions::consent::DynamicPermission>,
}

impl From<Vec<portable_config::definitions::consent::DynamicPermission>> for ConsentContent {
	fn from(value: Vec<portable_config::definitions::consent::DynamicPermission>) -> Self {
		Self { permissions: value }
	}
}
