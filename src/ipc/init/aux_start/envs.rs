/**
	Produce a map of variables to expose for the child

	Currently only XDG_ACTIVATION_TOKEN is accepted
*/
pub fn get() -> std::collections::HashMap<String, String> {
	let mut map = std::collections::HashMap::new();

	let envs = vec![
		"XDG_ACTIVATION_TOKEN",
	];

	for env in envs {
		match std::env::var(env) {
			Ok(v)	=> {
				match v.split_once("=") {
					Some((k, v))	=> {
						map.insert(k.into(), v.into());
					}
					None		=> {}
				}
			}
			Err(_)	=> {}
		}
	};
	map
}
