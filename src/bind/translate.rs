#[derive(thiserror::Error, Debug)]
pub enum TranslatePathError {
	#[error("Could not perform path translation: could not determine user home")]
	HomeNotFoundError,
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
