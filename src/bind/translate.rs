#[derive(thiserror::Error, Debug)]
pub enum TranslatePathError {
	#[error("Could not perform path translation: could not determine user home")]
	HomeNotFoundError,

	#[error("Could not perform path translation: {0:?} is not beneath {1:?}")]
	PathNotBeneathHomeError(String, String),
}

/*
	Translate the given input path beneath $HOME to the sandbox home
	Takes ownership of input value to avoid re-using.
*/
pub async fn home(origin: std::path::PathBuf) -> Result<std::path::PathBuf, TranslatePathError> {
	let home = {
		match std::env::home_dir() {
			Some(v)	=> {v}
			None	=> {
				return Err(TranslatePathError::HomeNotFoundError);
			}
		}
	};

	let mut home_iter = home.iter();
	let mut origin_iter = origin.iter();

	let mut path = std::path::PathBuf::new();

	loop {
		match home_iter.next() {
			Some(v)	=> {
				if Some(v) == origin_iter.next() {
					path.push(v);
				} else {
					return Err(
						TranslatePathError::PathNotBeneathHomeError(
							format!("{origin:?}"),
							format!("{home:?}"),
						),
					);
				}
			}
			None	=> {break;}
		}
	};

	loop {
		match origin_iter.next() {
			Some(v)	=> {
				path.push(v);
			}
			None	=> {
				return Ok(path);
			}
		}
	}

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
