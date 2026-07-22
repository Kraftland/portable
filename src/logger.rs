pub enum LogLevel {
	Debug,
	Info,
	Warn,
	Fatal,
}

struct LogMessage<T: std::fmt::Display> {
	pub level:		LogLevel,
	pub message:	T,
}

pub async fn logger<T> (
	mut log_rx: tokio::sync::mpsc::Receiver<LogMessage<T>>,
	stop_tx: tokio::sync::mpsc::Sender<crate::stop::StopFunc>,
)
	where T: std::fmt::Display
{
	let is_terminal = {
		let thread = tokio::task::spawn_blocking(
			|| get_termios(),
		)
		.await;

		let thread = match thread {
			Ok(v)	=> {v}
			Err(e)	=> {
				eprintln!("Could not spawn task: {e:#?}");
				panic!("{e:#?}")
			}
		};

		match thread {
			Some(v)	=> {

			let func = Box::new(move || {
							use std::os::fd::AsFd;
							match nix::sys::termios::tcsetattr(
								std::io::stdin().as_fd(),
								nix::sys::termios::SetArg::TCSANOW,
								&v,
							) {
								Ok(_)	=> {}
								Err(e)	=> {
									eprintln!("Could not restore console state: {e:#?}")
								}
							}
						});

				stop_tx.send(
					crate::stop::StopFunc {
						layer: crate::stop::FunctionLayer::Pre,
						function: func,
					},
				).await.expect("Could not request termination inhibitor");
				true
			}
			None	=> {
				eprintln!("Could not detect terminal status");
				false
			}
		}
	};


	let msg = tokio::select! {
		log_msg = log_rx.recv()	=> {
			match log_msg {
				Some(v)	=> v,
				None	=> {
					return;
				}
			}
		}
	};
}

fn get_termios() -> Option<nix::sys::termios::Termios> {
	use std::os::fd::AsFd;
	match nix::sys::termios::tcgetattr(std::io::stdin().as_fd()) {
		Ok(v)	=> {
			return Some(v);
		}
		Err(_)	=> {
			return None;
		}
	}
}

fn get_no_color_preference() -> bool {
	match std::env::var("NO_COLOR") {
		Ok(_)	=> {
			true
		}
		Err(_)	=> {
			false
		}
	}
}
