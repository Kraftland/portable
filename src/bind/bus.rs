pub mod rules;

/**
	The public struct proxy is used to define rules and sandboxing layer for xdg-dbus-proxy
*/
pub struct Proxy {
	sandbox:	crate::bind::types::BindRules,
	bus_access:	Vec<rules::BusAccessLevel>,
}
