#[derive(thiserror::Error, Debug)]
pub enum TranslatePathError {
	#[error("Could not perform path translation: could not determine user home")]
	HomeNotFoundError,

	#[error("Could not perform path translation: failed stripping path prefix: {0:?}")]
	StripPrefixError(String),
}


/*
	Delta designates the difference between the real $HOME and fake $HOME.
	It is the PREFIX after removing the original $HOME call.
	e.g. for an app with state directory test_Data,
		it's delta would be $XDG_DATA_DIR/test_Data
*/
pub struct Delta {
	path:	std::path::PathBuf,
}

impl Delta {
	/*
		This function involves 2 clones, it's better to cache the result somewhere
	*/
	pub async fn get(
		config:		&crate::config_definition::Config,
		xdg_dir:	&crate::xdg::XdgDirs,
	) -> Delta {
		let mut path = std::path::PathBuf::from(
			xdg_dir.data_home.clone()
		);
		path.push(config.metadata.state_directory.clone());
		Delta {
			path:	path
		}
	}
}

pub trait Translate {
	/*
		Delta is from struct Delta's get method
	*/
	async fn translate_home(self, delta: &Delta)	-> Result<std::path::PathBuf, TranslatePathError>;
}

impl Translate for std::path::PathBuf {
	async fn translate_home(self, delta: &Delta)	-> Result<std::path::PathBuf, TranslatePathError> {
		home(self, delta).await
	}
}

/*
	Translate the given input path beneath $HOME to the sandbox home
	Takes ownership of input value to avoid re-using.

	TODO: maybe implement it as a trait?

*/
pub async fn home(origin: std::path::PathBuf, delta: &Delta) -> Result<std::path::PathBuf, TranslatePathError> {
	let home = {
		match std::env::home_dir() {
			Some(v)	=> {v}
			None	=> {
				return Err(TranslatePathError::HomeNotFoundError);
			}
		}
	};

	let stripped_path = strip_prefix(origin, &home).await?;

	let mut ret = std::path::PathBuf::from(delta.path.clone());
	ret.push(stripped_path);
	Ok(ret)
}

async fn strip_prefix(
	origin: std::path::PathBuf,
	prefix: &std::path::PathBuf,
) -> Result<std::path::PathBuf, TranslatePathError> {
	let mut origin_iter = origin.iter();
	let mut stripped = std::path::PathBuf::new();
	// Iterate through prefix first to make sure it is really nested
	for pth in prefix.iter() {
		let origin_pth = match origin_iter.next() {
			Some(v)	=> {
				v
			}
			None	=> {
				return Err(
					TranslatePathError::StripPrefixError(
						format!("{origin:?} does not have prefix {prefix:?}"),
					),
				);
			}
		};
		if origin_pth == pth {
			continue;
		} else {
			return Err(
				TranslatePathError::StripPrefixError(
					format!("{origin:?} does not have prefix {prefix:?}"),
				),
			);
		}
	};

	loop {
		match origin_iter.next() {
			Some(v)	=> {
				stripped.push(v);
			}
			None	=> {
				break;
			}
		}
	};

	Ok(stripped)
}

async fn nested_in_home(origin: &std::path::PathBuf, home: &std::path::PathBuf) -> bool {
	let mut home_iter = home.iter();
	let mut origin_iter = origin.iter();
	loop {
		match home_iter.next() {
			Some(v)	=> {
				match origin_iter.next() {
					Some(val)	=> {
						if v == val {
							continue;
						} else {
							return false;
						}
					}
					None	=> {
						return false;
					}
				}
			}
			None	=> {break;}
		}
	}
	true
}
