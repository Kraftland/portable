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
pub struct ConsentContent {}
