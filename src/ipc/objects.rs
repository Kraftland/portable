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
	#[zbus(property)]
	async fn version(&self) -> u32 {
		crate::consts::DAEMON_VERSION
	}
}

struct AuxStart {
	started:	std::sync::atomic::AtomicBool,
}

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
		loop {
			if ! self.started.load(std::sync::atomic::Ordering::Relaxed) {
				tokio::time::sleep(std::time::Duration::from_millis(100)).await;
			} else {
				break;
			}
		}
		unimplemented!()
	}
}

struct Controller {
	stop_tx:	tokio::sync::mpsc::Sender<crate::stop::StopLevel>,
}

#[interface (name = "top.kimiblock.Portable.Controller")]
impl Controller {
	#[zbus(name = "Stop")]
	async fn stop(&self) {
		self.stop_tx.send(crate::stop::StopLevel::Normal)
			.await
			.expect("Could not send stop signal");
	}
}
