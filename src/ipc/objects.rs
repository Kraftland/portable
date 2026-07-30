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

struct AuxStart;

#[interface (name = "top.kimiblock.portable.AuxStart")]
impl AuxStart {
	#[zbus(name = "RequestStart1")]
	async fn start_v1 (
		&self,
		custom_target:	bool,
		target_exec:	String,
		args_append:	bool,
		arguments:	Vec<String>,
		extra_files:	std::collections::HashMap<String, String>,
		envs:		std::collections::HashMap<String, String>,
	) -> zbus::zvariant::OwnedFd {
		unimplemented!()
	}
}
