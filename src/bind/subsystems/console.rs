pub fn is_terminal() -> bool {
	match get_termios() {
		Some(_)	=> {
			true
		}
		None	=> {
			eprintln!("Could not detect terminal status");
			false
		}
	}
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
