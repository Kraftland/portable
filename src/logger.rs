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


