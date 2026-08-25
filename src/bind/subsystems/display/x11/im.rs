pub enum InputMethodKind {
	IBus,
	Fcitx,
	Gcin,
	Unknown,
}

pub async fn detect_kind() -> InputMethodKind {
	let check_envs = vec![
		"XMODIFIERS",
		"INPUT_METHOD",
		"QT_IM_MODULE",
		"GTK_IM_MODULE",
	];

	for env in check_envs {
		let var = match std::env::var(env) {
			Ok(v)	=> {v}
			Err(_)	=> {continue;}
		};

		if var.contains("ibus") {
			return InputMethodKind::IBus;
		} else if var.contains("fcitx") {
			return InputMethodKind::Fcitx;
		} else if var.contains("gcin") {
			return InputMethodKind::Gcin;
		}
	};

	InputMethodKind::Unknown
}
