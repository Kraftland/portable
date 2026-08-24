/**
	Produce a map of variables to expose for the child

	Currently only XDG_ACTIVATION_TOKEN is accepted
*/
pub fn get() -> std::collections::HashMap<String, String> {
	let mut map = std::collections::HashMap::new();

	let envs = vec![
		"XDG_ACTIVATION_TOKEN",
		"TERM",
	];

	for env in envs {
		match std::env::var(&env) {
			Ok(v)	=> {
				map.insert(env.into(), v.into());
			}
			Err(_)	=> {}
		}
	};
	map
}
