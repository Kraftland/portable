use serde::{Deserialize, Deserializer};
use serde::de::Error;

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Config {

}

#[derive(Debug, Deserialize)]
pub struct Metadata {
	#[serde(alias = "appID")]
	// Check needed
	pub sandbox_id:		String,
	#[serde(alias = "friendlyName")]
	pub display_name:	String,
	#[serde(alias = "stateDirectory")]
	pub state_directory:	String,

	pub config_version:	usize,
}

#[derive(Debug, Deserialize)]
pub struct Exec {
	#[serde(alias = "target")]
	pub target:		String,
	#[serde(alias = "arguments")]
	pub arguments:		Vec<String>,
	#[serde(alias = "overlay")]
	pub overlay:		bool,
}

#[derive(Debug, Deserialize)]
pub struct BusExec {
	pub enable:		bool,
	#[serde(alias = "target")]
	pub target:		String,
	#[serde(alias = "arguments")]
	pub arguments:		Vec<String>,
	#[serde(alias = "overlay")]
	pub overlay:		bool,
}

#[derive(Debug,  Deserialize)]
pub struct ProcMgmt {
	#[serde(default = "default_background")]
	pub background:		bool,
}

fn default_background() -> bool {
	true
}

#[derive(Debug, Deserialize)]
pub struct SysMgmt {
	#[serde(alias = "inhibitSuspend")]
	pub allow_inhibit:	bool,

	#[serde(alias = "inhibitOnBehalf")]
	pub conduct_inhibit:	bool,

	#[serde(alias = "uclamp")]
	#[serde(default = "default_uclamp")]
	pub uclamp_max:		u16,

	#[serde(alias = "deviceAllow")]
	#[serde(deserialize_with = "deserialise_device_allow")]
	pub device_allow:	Vec<DeviceAllow>,
}

fn default_uclamp () -> u16 {
	100
}

#[derive(Debug)]
pub enum DeviceAllow {
	DiscreteGPU,
	Input,
	Camera,
	Kvm,
}

fn deserialise_device_allow <'de, D> (deserialiser: D) -> Result<Vec<DeviceAllow>, D::Error>
	where
		D: Deserializer<'de>,
{
	let mut ret = vec![];
	let raw_allow = Vec::<String>::deserialize(deserialiser)?;
	for arg in raw_allow.iter() {
		match arg.as_str() {
			"dgpu"	=>	{
				ret.push(
					DeviceAllow::DiscreteGPU,
				);
			}
			"input"	=>	{
				ret.push(
					DeviceAllow::Input,
				);
			}
			"camera"=>	{
				ret.push(
					DeviceAllow::Camera
				);
			}
			"kvm"	=>	{
				ret.push(
					DeviceAllow::Kvm,
				);
			}
			_	=>	{
				return Err(D::Error::custom(
					"Invalid device_allow argument"
				));
			}
		}
	};
	Ok(ret)
}

#[derive(Debug, Deserialize)]
pub struct Network {
	#[serde(alias = "enable")]
	pub allow_network:	bool,
	#[serde(alias = "filter")]
	pub enable_filter:	bool,
	#[serde(alias = "filterDest")]
	pub block_dest:		Vec<NetworkFilterTarget>,
}

#[derive(Debug,Deserialize)]
#[serde(untagged)]
pub enum NetworkFilterTarget {
	IPAddr (std::net::IpAddr),
	DomainOrPrivate (String)
}

#[derive(Debug, Deserialize)]
pub struct Privacy {
	pub lockdown:		bool,

	#[serde(alias = "x11")]
	pub x11_compat:		bool,

	#[serde(alias = "classicNotifications")]
	pub classic_notif:	bool,

	#[serde(alias = "pipeWire")]
	pub pipewire:		bool,
}

#[derive(Debug, Deserialize)]
pub struct Advanced {
	#[serde(alias = "zink")]
	pub use_zink:		bool,

	#[serde(alias = "qt5Compat")]
	#[serde(default = "default_qt5_compat")]
	pub qt5_compat:		bool,

	#[serde(alias = "mprisName")]
	pub mpris_names:	Vec<String>,

	#[serde(alias = "trayWake")]
	pub tray_wake:		bool,

	#[serde(alias = "kDEStatus")]
	pub allow_kde_status:	bool,

	#[serde(alias = "flatpakInfo")]
	#[serde(default = "default_flatpak_env")]
	pub flatpak_env:	bool,

	#[serde(alias = "debugging")]
	pub allow_debug:	bool,
}

fn default_qt5_compat () -> bool {
	true
}

fn default_flatpak_env () -> bool {
	true
}
