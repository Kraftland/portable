/**
	Generate argv0 and cmdline for ExecStartEx

	Borrows the Spawn struct
*/
pub async fn cmdline(spawn: std::sync::Arc<crate::spawn::Spawn>) -> (String, Vec<String>) {
	use crate::bind::types::ToCmdline;

	let argv0 = String::from("bwrap");

	let mut cmd = vec![
		String::from("bwrap"),
		String::from("--new-session"),
		String::from("--unshare-all"),
		String::from("--share-net"),
	];
	cmd.extend(
		spawn
			.fs_rules
			.to_cmdline()
			.await
	);

	cmd.push("--".into());
	cmd.push("/usr/lib/portable/helper/helper".into());
	cmd.push(spawn.config.metadata.sandbox_id.to_string());

	(argv0, cmd)
}
