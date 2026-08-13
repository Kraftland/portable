use zbus::interface;

pub struct Info;

#[interface (name = "top.kimiblock.portable.Info")]
impl Info {
	#[zbus(name = "GetInfo")]
	pub async fn get_info (&self) -> Vec<String> {
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
	#[zbus(property)]
	pub async fn version(&self) -> u32 {
		crate::consts::DAEMON_VERSION
	}
}

pub struct Controller {
	pub stop_tx:	tokio::sync::mpsc::Sender<crate::stop::StopLevel>,
}

#[interface (name = "top.kimiblock.Portable.Controller")]
impl Controller {
	#[zbus(name = "Stop")]
	pub async fn stop(&self) {
		self.stop_tx.send(crate::stop::StopLevel::Normal)
			.await
			.expect("Could not send stop signal");
	}
}
