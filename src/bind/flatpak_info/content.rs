/**
	Generate the content for flatpak-info file
*/
pub async fn generate(
	config:		std::sync::Arc<crate::config::config_definition::Config>,
	instance_id:	std::sync::Arc<String>,
	xdg:		std::sync::Arc<crate::xdg::XdgDirs>,
) -> String {
	let mut content = String::new();

	{
		content.push_str("[Application]");
		content.push_str("\n");
		content.push_str("name=");
		content.push_str(&config.metadata.sandbox_id);
		content.push_str("\n");
		content.push_str("runtime=runtime/org.kraftland.host/x86_64/12252019");
		content.push_str("\n");
	};
	{
		content.push_str("[Instance]");
		content.push_str("\n");
		content.push_str("instance-id=");
		content.push_str(&instance_id);
		content.push_str("\n");
		content.push_str("instance-path=");
		let state_dir = {
			let mut path = xdg.data_home.to_path_buf();
			path.push(&config.metadata.state_directory);
			path
		};
		content.push_str(&state_dir.to_string_lossy());
		content.push_str("\n");
		content.push_str("app-path=/usr");
		content.push_str("\n");
		content.push_str("app-commit=");
		content.push_str("e894d778d380b02cce56cd42e326b244df8bbf298f06a1e3573a2f32754a0207");
		content.push_str("\n");
		content.push_str("runtime-path=/");
		content.push_str("\n");
		content.push_str("runtime-commit=");
		content.push_str("6087f25c76665f35dc9790e60a89f1af2481de9b5c35ee71b9b16a86f388bf3c");
		content.push_str("\n");
		content.push_str("branch=stable");
		content.push_str("\n");
		content.push_str("arch=x86_64");
		content.push_str("\n");
		content.push_str("flatpak-version=1.19.0"); // it was 1.16.0
		content.push_str("\n");
		content.push_str("session-bus-proxy=true");
		content.push_str("\n");
		content.push_str("system-bus-proxy=true");
		content.push_str("\n");
		content.push_str("extra-args=--usb-list=;--filesystem=");
		content.push_str(&state_dir.to_string_lossy());
		content.push_str(";");
		content.push_str("\n");
	};
	{
		content.push_str("[Context]");
		content.push_str("\n");
		content.push_str("shared=network;");
		content.push_str("\n");
		content.push_str("sockets=x11;wayland;");
		content.push_str("\n");
		content.push_str("devices=dri;");
		content.push_str("\n");
	};

	{
		content.push_str("[USB Devices]");
		content.push_str("\n");
		content.push_str("enumerable-devices=all;");
		content.push_str("\n");
	};

	content
}
