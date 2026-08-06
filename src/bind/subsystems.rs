
/**
	A generic trait for other subsystems to implement binding generation

	Portable's bind rule generation system is divided to multiple subsystems. Each of them may
	implement different functions and are generally controlled via Cargo feature switches.
*/
pub trait GenerateBind {
	fn bind(self) -> impl std::future::Future<Output = super::types::BindRules> + Send;
}
