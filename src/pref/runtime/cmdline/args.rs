/**
	Parse the command line options

	Note that --actions flag is dropped, but it does not actually affect backwards compatibility

	This returns a initialised RuntimeOpts struct, thus must be used as default option.
*/
pub async fn parse(logger: crate::logger::LogSender)
-> Result<crate::pref::runtime::options::RuntimeOpts, CmdlineError> {
	use crate::pref::runtime::options::FileExposurePreference;
	use crate::pref::runtime::options::Action;


	let mut file_forwarding: bool = false;
	let mut bus_activate: bool = false;
	let mut expose_files: Vec<FileExposurePreference> = vec![];
	let mut start_mode: Action = Action::Normal;
	let mut debug_shell: bool = false;
	let mut application_args = vec![];

	let mut skip_counter = 0;

	let args: Vec<String> = std::env::args().collect();

	for (i, arg) in args.iter().enumerate().skip(1) {
		if skip_counter > 0 {
			skip_counter -= 1;
			continue;
		}

		match arg.as_str() {
			"--"					=> {
				if let Some(v) = args.get(i + 1..) {
					application_args.extend(v.to_vec());
				}
			}
			"--file-forwarding" | "--forward-file"	=> {
				file_forwarding = true
			}
			"--dbus-activation"			=> {
				bus_activate = true
			}
			"--expose"				=> {
				skip_counter = 2;
				let source = {
					match args.get(i + 1) {
						Some(v)	=> std::path::PathBuf::from(v),
						None	=> {
							return Err(CmdlineError::UnfinishedExpose);
						}
					}
				};

				let dest = match args.get(i + 2) {
					Some(v)	=> v,
					None	=> {
						return Err(CmdlineError::UnfinishedExpose)
					}
				};

				let (dest, class) = handle_expose_path(dest);

				if ! check_absolute(&source, &dest) {
					return Err(CmdlineError::NonAbsolutePath);
				}

				expose_files.push(
					FileExposurePreference::MountPath {
						host: std::path::PathBuf::from(source),
						dest: std::path::PathBuf::from(dest),
						class: class,
					}
				);
			}
			"quit" | "--quit"			=> {
				start_mode = Action::Quit
			}
			"debug-shell" | "--debug-shell"		=> {
				debug_shell = true
			}
			"share-file" | "share-files" | "--share-file" | "--share-files"
								=> {
				start_mode = Action::ShareFile
			}
			"share-directories" | "share-directory" | "--share-directory"
								=> {
				start_mode = Action::ShareDir
			}
			"opendir" | "--opendir" | "home" | "openhome"
								=> {
				start_mode = Action::OpenHome
			}
			"reset-document" | "reset-documents" | "--reset-document" |
			"--revoke-permission" | "--revoke-permissions"
								=> {
				start_mode = Action::ResetDocs
			}
			"f5aaebc6-0014-4d30-beba-72bce57e0650"	=> {
				return Err(CmdlineError::UnsafeError);
			}
			&_					=> {
				let _ = logger.send(
					crate::logger::LogMessage {
						level: crate::logger::LogLevel::Warn,
						message: format!("Unrecognised argument: {arg}"),
					},
				).await;
			}
		}
	};

	if file_forwarding {
		expose_files.extend(
			handle_file_forwarding(&application_args).await
		);
	}

	Ok(
		crate::pref::runtime::options::RuntimeOpts {
			file_expose: 	expose_files,
			action:		start_mode,
			app_args:	application_args,
			bus_activation:	bus_activate,
			debug_shell:	debug_shell,
		},
	)
}

fn check_absolute(source: &std::path::PathBuf, dest: &std::path::PathBuf) -> bool {
	match source.as_path().is_absolute() {
		true	=> {}
		false	=> {
			return false
		}
	};

	match dest.as_path().is_absolute() {
		true	=> {}
		false	=> {
			return false
		}
	};

	true
}

async fn handle_file_forwarding(app_args: &Vec<String>) ->
Vec<crate::pref::runtime::options::FileExposurePreference> {
	use crate::pref::runtime::options::FileExposurePreference;
	let mut ret = vec![];

	for arg in app_args {
		let path = std::path::PathBuf::from(arg);
		if path.as_path().is_absolute() {
			ret.push(FileExposurePreference::Passthrough { host: path });
		}
	};

	ret
}

/**
	Handle a command line expose flag

	ro: means read only, dev: means device
*/
fn handle_expose_path(cmd: &str) -> (std::path::PathBuf, crate::bind::types::BindType) {
	use std::path::PathBuf;
	use crate::bind::types::BindType;
	match cmd.strip_prefix("ro:") {
		Some(v)	=> {
			return (PathBuf::from(v), BindType::ReadOnly);
		}
		None	=> {}
	};

	match cmd.strip_prefix("dev:") {
		Some(v)	=> {
			return (PathBuf::from(v), BindType::Device);
		}
		None	=> {}
	};

	(PathBuf::from(cmd), BindType::ReadWrite)
}

#[derive(thiserror::Error, Debug)]
pub enum CmdlineError {
	#[error("Unfinished --expose flag: requires both source and destination")]
	UnfinishedExpose,

	#[error("Non-absolute path in --expose flag")]
	NonAbsolutePath,

	#[error("Unsafe mode is removed after legacy branch deprecation")]
	UnsafeError,
}
