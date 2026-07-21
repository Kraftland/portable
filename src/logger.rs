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
	mut log_rx: tokio::sync::mpsc::Receiver<LogMessage<T>>
)
	where T: std::fmt::Display
{
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

fn is_terminal() -> (bool, Option<nix::sys::termios::Termios>) {
	use std::os::fd::AsFd;
	match nix::sys::termios::tcgetattr(std::io::stdin().as_fd()) {
		Ok(v)	=> {
			return (true, Some(v));
		}
		Err(_)	=> {
			return (false, None);
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
