pub async fn mask_spawn() -> Option<crate::bind::types::BindRule> {
	if tokio::fs::try_exists("/usr/lib/flatpak-xdg-utils/flatpak-spawn").await.unwrap_or(false) {
		Some(crate::bind::types::BindRule::Path {
			source: "/usr/lib/portable/overlay-usr/flatpak-spawn".into(),
			dest: "/usr/lib/flatpak-xdg-utils/flatpak-spawn".into(),
			class: crate::bind::types::BindType::ReadOnly,
		})
	} else {
		None
	}
}
