use crate::bind::bus::Proxy;

#[derive(thiserror::Error, Debug)]
pub enum StartProxyError {
	#[error("Could not start D-Bus proxy: {0:#?}")]
	SpawnError(std::io::Error),

	#[error("Could not start D-Bus proxy: mapping file descriptors failed: {0:#?}")]
	FDError(command_fds::FdMappingCollision),
}

impl Proxy {
	pub async fn start(self)	-> Result<(), StartProxyError> {
		use command_fds::{CommandFdExt, FdMapping};
		use crate::bind::types::ToCmdline;

		let mut builder = tokio::process::Command::new("bwrap");
		builder.stdin(std::process::Stdio::null());

		{
			builder.stdout(std::process::Stdio::null());
		};

		#[cfg(debug_assertions)]
		{
			builder.stdout(std::process::Stdio::inherit())
		};

		builder.kill_on_drop(false);

		let mut cmdline = vec!["--unshare-all".to_string()];


		match self.json_status_file {
			Some(v)	=> {
				builder.fd_mappings(
					vec![
						FdMapping {
							parent_fd:	v,
							child_fd:	25,
						}
					],
				)
					.map_err(StartProxyError::FDError)
					?;
				cmdline.push("--json-status-fd".into());
				cmdline.push("25".into());
			}
			None	=> {}
		};

		{
			let sandbox_cmd = self.sandbox.to_cmdline().await;
			for cmd in sandbox_cmd {
				cmdline.push(cmd);
			};
		};

		{
			cmdline.push("--".into());
			cmdline.push("xdg-dbus-proxy".into());
			cmdline.push(self.bus_address.clone());
			cmdline.push(self.proxy_address);
			cmdline.push("--filter".into());

			if self.sloppy_names {
				cmdline.push("--sloppy-names".into());
			};

			let bus_args = self.bus_access.to_cmdline().await;
			for arg in bus_args {
				cmdline.push(arg);
			};
		};

		builder.args(cmdline);

		let mut child = builder
			.spawn()
			.map_err(StartProxyError::SpawnError)
			?;

		let _ = self.logger.send(
			crate::logger::LogMessage {
				level: crate::logger::LogLevel::Debug,
				message: format!("Started D-Bus proxy for {}", self.bus_address),
			},
		).await;

		match self.bind_lifetime {
			Some(v)	=> {
				tokio::spawn(async move {
					let stop_channel = v;
					let result = child.wait().await;
					match result {
						Ok(_)	=> {
							let _ = self.logger.send(
								crate::logger::LogMessage {
									level: crate::logger::LogLevel::Debug,
									message: format!("D-Bus proxy exited"),
								},
							).await;
							stop_channel
								.send(crate::stop::StopLevel::Normal)
								.await
								.expect("Could not send stop signal");
						}
						Err(e)	=> {
							let _ = self.logger.send(
								crate::logger::LogMessage {
									level: crate::logger::LogLevel::Fatal,
									message: format!(
										"D-Bus proxy failed: {e:#?}",
									),
								}
							).await;
							stop_channel
								.send(crate::stop::StopLevel::Error(1))
								.await
								.expect("Could not send stop signal");
						}
					}
				});
				Ok(())
			}
			None	=> {
				Ok(())
			}
		}
	}
}
