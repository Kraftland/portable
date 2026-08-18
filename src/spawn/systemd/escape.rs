pub type UnitName = String;

/**
	Escape the String into a system unit name
*/
pub fn unit_name<T>(name: T) -> UnitName
	where T: ToString
{
	let original_name = name.to_string();

	zbus_systemd::unit_name_escape(&original_name).to_string()
}
