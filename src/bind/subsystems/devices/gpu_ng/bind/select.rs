/**
	The public function select decides a list of GPUs to expose inside said sandbox.

	It should also convert them into bind rules, including vendor workarounds.

	It accepts a collection of GPU info and perform various decisions, some even be heuristics
		to select a primary GPU (or GPUs).

	If game mode is enabled, then we short circuit to all GPUs.
*/
pub async fn select<T>(
	devices:	T,
	all_gpus:	bool,
)
->
	crate::bind::types::BindRules
where
	T:	IntoIterator<Item = super::GPUInfo>,
{
	let mut gpu_map = vec![];

	if all_gpus {
		for dev in devices {
			gpu_map.push((dev, true));
		};
	} else {
		for dev in devices {
			if dev.boot_display {
				gpu_map.push((dev, true));
			} else {
				gpu_map.push((dev, false));
			};
		};
	};

	unimplemented!()
}
