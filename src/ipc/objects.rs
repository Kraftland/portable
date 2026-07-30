use zbus::interface;

struct Info;

#[interface (name = "top.kimiblock.portable.Info")]
impl Info {
	#[zbus(name = "GetInfo")]
	async fn get_info (&self) -> Vec<String> {
		vec![
			{
				let mut string = String::from("Daemon version: ");
				string.push_str(&crate::consts::DAEMON_VERSION.to_string());
				string
			},
			String::from("Not implemented")
		]
	}

	#[zbus(name = "Version")]
	async fn version(&self) -> u32 {
		crate::consts::DAEMON_VERSION
	}
}
