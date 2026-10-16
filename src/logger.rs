mod config;

/**
	The messages module holds all possible logging messages.

	They are defined in four different categories: debug, info, warn, critical.

	The central LogMessage Enum holds aforementioned categories as enum variants, and those four
	then hold the actual message.

	Every variant must implement the std::fmt::Display trait.
*/
pub mod messages;

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

pub type LogSender = tokio::sync::mpsc::Sender<messages::LogMessage>;

pub async fn logger (
	mut log_rx:	tokio::sync::mpsc::Receiver<messages::LogMessage>,
	stop:		std::sync::Arc<crate::stop::Stop>,
)
{

	/*
		Core of the new race-free shutdown design.

		The stop worker will call master_token.cancel, while outside we wait for slave_token
		before closing channels.
	*/
	let (master_token, slave_token) = {
		let token = tokio_util::sync::CancellationToken::new();
		let slave = token.child_token();
		(
			token,
			slave,
		)
	};
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

	{
		let stop_cancel = stop.pre_parent.child_token();
		stop.stop_funcs.send(
			crate::stop::StopMessage::Prepare {
				task: tokio::spawn(async move {
					stop_cancel.cancelled().await;

					master_token.cancel();

					Ok(())
				}),
			}
		).unwrap_or_else(
			|e| {
				eprintln!("Could not contact the stop worker: {e}");
				()
			}
		)
	};

	while let Some(msg) = log_rx.recv().await {
		match msg {
			messages::LogMessage::Debug(_)	=> {
				#[cfg(debug_assertions)]
				println!("{}\t{}", debug_fmt, msg);
			}
			messages::LogMessage::Info(_)	=> {
				println!("{}\t\t{}", info_fmt, msg);
			}
			messages::LogMessage::Warn(_)	=> {
				eprintln!("{}\t\t{}", warn_fmt, msg);
			}
			messages::LogMessage::Fatal(_)	=> {
				eprintln!("{}\t{}", fatal_fmt, msg);
			}
		};

		if slave_token.is_cancelled() {
			log_rx.close();
		}
	};
}
