pub enum SessionType {
	Wayland,
	X11,
	Unknown,
}

pub async fn detect() -> SessionType {
	match std::env::var("XDG_SESSION_TYPE").unwrap_or(String::new()).as_str() {
		"wayland"	=> {
			SessionType::Wayland
		}
		"x11"		=> {
			SessionType::X11
		}
		_		=> {
			SessionType::Unknown
		}
	}
}
