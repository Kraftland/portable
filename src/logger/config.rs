impl super::LoggingConfig {
	pub fn get() -> Self {
		match is_terminal() {
			true	=>	{}
			false	=>	{
				return Self::Plain;
			}
		};
		match is_colour() {
			true	=> {}
			false	=> {
				return Self::Plain;
			}
		};

		match is_pups_day() {
			true	=> {
				Self::Console { colour: super::ColourVariant::Special }
			}
			false	=> {
				Self::Console { colour: super::ColourVariant::Normal }
			}
		}
	}
}



/**
	Use the nix crate to get termios struct from stdin
*/
fn is_terminal() -> bool {
	use std::os::fd::AsFd;

	match nix::sys::termios::tcgetattr(std::io::stdin().as_fd()) {
		Ok(_)	=> true,
		Err(_)	=> false,
	}
}

/**
	Does the client want coloured output?
*/
fn is_colour() -> bool {
	match std::env::var("NO_COLOR") {
		Ok(v)	=> {
			if v.is_empty() {
				true
			} else {
				false
			}
		}
		Err(_)	=> {
			false
		}
	}
}

fn is_pups_day() -> bool {
	let time = jiff::Zoned::now();
	if time.month() == 12 && time.day() == 25 {
		true
	} else {
		false
	}
}
