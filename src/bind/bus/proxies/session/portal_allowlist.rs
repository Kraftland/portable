pub async fn get_allowed_portals(inhibit: bool) -> Vec<String> {
	let rule = vec![
		"Account".to_string(),
		"Camera".into(),
		"Clipboard".into(),
		"Email".into(),
		"FileChooser".into(),
		"Location".into(),
		"InputCapture".into(),
		"MemoryMonitor".into(),
		"NetworkMonitor".into(),
		"Notification".into(),
		"OpenURI".into(),
		"PowerProfileMonitor".into(),
		"Print".into(),
		"ProxyResolver".into(),
		"RemoteDesktop".into(),
		"ScreenCast".into(),
		"Screenshot".into(),
		"Secret".into(),
		"Settings".into(),
		"Trash".into(),
		"Usb".into(),
		"Wallpaper".into(),
		"GlobalShortcuts".into(),
	];

	if inhibit {
		rule.push("Inhibit".into());
	}

	rule
}
