/**
	The public function select decides a list of GPUs to expose inside said sandbox.

	It accepts a collection of GPU info and perform various decisions, some even be heuristics
		to select a primary GPU (or GPUs).

	If game mode is enabled, then we short circuit to all GPUs.
*/
pub async fn select<T>(
	devices:	T,
	all_gpus:	bool,
)
->
	Vec<super::GPUInfo>
where
	T:	IntoIterator<Item = super::GPUInfo>,
{
	if all_gpus {
		return devices.into_iter().collect();
	};

	let mut ret = vec![];

	for device in devices {
		if device.boot_display {
			ret.push(device);
		}
	};

	ret
}
