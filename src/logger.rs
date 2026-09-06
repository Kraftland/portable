mod config;

/**
	LoggingConfig designates the initial configuration for logging thread.

	It has a function get() implemented by config module.
*/
#[derive(Debug)]
pub enum LoggingConfig {
	Console {
		colour:	ColourVariant,
	},
	Plain {
		reason:	config::PlainReason,
	},
}

#[derive(Debug)]
pub enum ColourVariant {
	Normal,
	Special,
}

pub enum LogLevel {
	Debug,
	Info,
	Warn,
	Fatal,
}

pub struct LogMessage {
	pub level:		LogLevel,
	pub message:		String,
}

pub type LogSender = tokio::sync::mpsc::Sender<LogMessage>;

pub async fn logger (
	mut log_rx:	tokio::sync::mpsc::Receiver<LogMessage>,
	stop_token:	tokio_util::sync::CancellationToken,
)
{
	let logging_config = LoggingConfig::get();

	let (
		debug_fmt,

		info_fmt,
		warn_fmt,
		fatal_fmt,
	) = {
		match &logging_config {
			LoggingConfig::Plain { reason: _ }
						=> {
				(
					"[Debug]:",
					"[Info]:",
					"[Warn]:",
					"[Fatal]:",
				)
			}
			LoggingConfig::Console { colour }
						=> {
				match colour {
					ColourVariant::Normal	=> {
						(
							"\x1b[38;2;125;241;118m[Debug]\x1b[0m:",
							"\x1b[38;2;119;222;250m[Info]\x1b[0m:",
							"\x1b[38;2;255;209;59m[Warn]\x1b[0m:",
							"\x1b[38;2;255;0;0m[Fatal]\x1b[0m:",
						)
					}
					ColourVariant::Special	=> {
						(
							"\x1b[38;2;213;161;115m[Debug]\x1b[0m:",
							"\x1b[38;2;213;161;115m[Info]\x1b[0m:",
							"\x1b[38;2;213;161;115m[Warn]\x1b[0m:",
							"\x1b[38;2;255;0;0m[Fatal]\x1b[0m:",
						)
					}
				}
			}
		}
	};

	#[cfg(debug_assertions)]
	println!("{}\tUsing logger configuration: {:#?}", &debug_fmt, &logging_config);

	loop {
		let msg = tokio::select! {
			biased;
			log_msg = log_rx.recv()			=> {
				match log_msg {
					Some(v)	=> v,
					None	=> {
						return;
					}
				}
			}
			_	= stop_token.cancelled()	=> {
				return ;
			}
		};

		match msg.level {
			LogLevel::Debug	=> {
				#[cfg(debug_assertions)]
				println!("{}\t{}", debug_fmt, msg.message);
			}
			LogLevel::Info => {
				println!("{}\t\t{}", info_fmt, msg.message);
			}
			LogLevel::Warn => {
				eprintln!("{}\t\t{}", warn_fmt, msg.message);
			}
			LogLevel::Fatal => {
				eprintln!("{}\t{}", fatal_fmt, msg.message);
			}
		}
	}


}
