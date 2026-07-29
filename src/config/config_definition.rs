use serde::{Deserialize, Deserializer};
use serde::de::Error;

fn default_false()		-> bool {false}

fn default_true()		-> bool {true}

fn default_empty_vec_string()	-> Vec<String> {vec![]}
fn default_empty_vec_network()	-> Vec<NetworkFilterTarget> {vec![]}

fn default_empty_string()	-> String {String::new()}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Config {
	pub metadata:		Metadata,

	pub exec:		Exec,

	#[serde(alias = "busActivation")]
	pub dbus_activation:	BusExec,

	#[serde(alias = "processes")]
	pub process:		ProcMgmt,

	pub system:		SysMgmt,

	pub network:		Network,

	pub privacy:		Privacy,

	pub advanced:		Advanced,
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

	#[serde(default = "default_config_version")]
	pub config_version:	usize,
}

fn default_config_version () -> usize {0}

#[derive(Debug, Deserialize)]
pub struct Exec {
	#[serde(alias = "target")]
	pub target:		String,

	#[serde(alias = "arguments")]
	#[serde(default = "default_empty_vec_string")]
	pub arguments:		Vec<String>,

	#[serde(alias = "overlay")]
	#[serde(default = "default_false")]
	pub overlay:		bool,
}

#[derive(Debug, Deserialize)]
pub struct BusExec {
	#[serde(default = "default_false")]
	pub enable:		bool,
	#[serde(alias = "target")]
	#[serde(default = "default_empty_string")]
	pub target:		String,
	#[serde(alias = "arguments")]
	#[serde(default = "default_empty_vec_string")]
	pub arguments:		Vec<String>,
	#[serde(alias = "overlay")]
	#[serde(default = "default_false")]
	pub overlay:		bool,
}

#[derive(Debug,  Deserialize)]
pub struct ProcMgmt {
	#[serde(default = "default_true")]
	pub background:		bool,
}

#[derive(Debug, Deserialize)]
#[serde(default)]
pub struct SysMgmt {
	#[serde(alias = "inhibitSuspend")]
	pub allow_inhibit:	bool,

	#[serde(alias = "inhibitOnBehalf")]
	pub conduct_inhibit:	bool,

	pub uclamp_max:		u16,

	#[serde(alias = "deviceAllow")]
	#[serde(deserialize_with = "deserialise_device_allow")]
	pub device_allow:	Vec<DeviceAllow>,
}

impl Default for SysMgmt {
	fn default() -> Self {
		Self {
			allow_inhibit: false,
			conduct_inhibit: false,
			uclamp_max: 100,
			device_allow: vec![],
		}
	}
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
	#[serde(default = "default_false")]
	pub allow_network:	bool,
	#[serde(alias = "filter")]
	#[serde(default = "default_false")]
	pub enable_filter:	bool,
	#[serde(alias = "filterDest")]
	#[serde(default = "default_empty_vec_network")]
	pub block_dest:		Vec<NetworkFilterTarget>,
}

#[derive(Debug, Deserialize)]
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

impl Default for Privacy {
	fn default() -> Self {
		Self {
			lockdown: false,
			x11_compat: false,
			classic_notif: false,
			pipewire: false,
		}
	}
}

#[derive(Debug, Deserialize)]
#[serde(default)]
pub struct Advanced {
	#[serde(alias = "zink")]
	pub use_zink:		bool,

	#[serde(alias = "qt5Compat")]
	pub qt5_compat:		bool,

	#[serde(alias = "mprisName")]
	pub mpris_names:	Vec<String>,

	#[serde(alias = "trayWake")]
	pub tray_wake:		bool,

	#[serde(alias = "kDEStatus")]
	pub allow_kde_status:	bool,

	#[serde(alias = "flatpakInfo")]
	pub flatpak_env:	bool,

	#[serde(alias = "debugging")]
	pub allow_debug:	bool,
}

impl Default for Advanced {
	fn default() -> Self {
		Self {
			use_zink: false,
			qt5_compat: true,
			mpris_names: vec![],
			tray_wake: false,
			allow_kde_status: false,
			flatpak_env: true,
			allow_debug: false,
		}
	}
}
