use serde::{Deserialize, Deserializer};
use serde::de::Error;

fn default_false()		-> bool {false}

fn default_empty_vec_string()	-> Vec<String> {vec![]}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Config {
	pub metadata:		Metadata,

	pub exec:		Exec,

	#[serde(default)]
	#[serde(alias = "busActivation")]
	pub dbus_activation:	BusExec,

	#[serde(default)]
	pub system:		SysMgmt,

	#[serde(default)]
	pub network:		Network,

	#[serde(default)]
	pub privacy:		Privacy,

	#[serde(default)]
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
}

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
#[serde(default)]
pub struct BusExec {
	pub enable:		bool,
	#[serde(alias = "target")]
	pub target:		String,
	#[serde(alias = "arguments")]
	pub arguments:		Vec<String>,
}

impl Default for BusExec {
	fn default() -> Self {
		Self {
			enable: false,
			target: String::new(),
			arguments: vec![],
		}
	}
}

#[derive(Debug, Deserialize)]
#[serde(default)]
pub struct SysMgmt {
	#[serde(alias = "inhibitSuspend")]
	pub allow_inhibit:	bool,

	#[serde(alias = "inhibitOnBehalf")]
	pub conduct_inhibit:	bool,

	pub uclamp_max:		u32,

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

#[derive(Debug, Clone, Copy)]
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
#[serde(default)]
pub struct Network {
	#[serde(alias = "enable")]
	pub allow_network:	bool,
	#[serde(alias = "filter")]
	pub enable_filter:	bool,
	#[serde(alias = "filterDest")]
	pub block_dest:		Vec<NetworkFilterTarget>,
}

impl Default for Network {
	fn default() -> Self {
		Self {
			allow_network: false,
			enable_filter: false,
			block_dest: vec![],
		}
	}
}

#[derive(Debug, Deserialize)]
#[serde(untagged)]
pub enum NetworkFilterTarget {
	IPAddr (std::net::IpAddr),
	DomainOrPrivate (String)
}

#[derive(Debug, Deserialize)]
#[serde(default)]
pub struct Privacy {
	pub lockdown:		Lockdown,

	#[serde(alias = "x11")]
	pub x11_compat:		bool,

	#[serde(alias = "classicNotifications")]
	pub classic_notif:	bool,

	pub push_notification:	bool,
}

#[derive(Debug, Deserialize)]
#[serde(untagged)]
pub enum Lockdown {
	Global	{ enable: bool },
	FineGrained {
		seccomp:	bool,
		landlock:	bool,
	},
}

pub struct LockdownOptions {
	pub seccomp_whitelist:	bool,
	pub landlock:		bool,
}

impl Lockdown {
	/**
		Translate the current Lockdown enum to a fine-grained lockdown feature enablement.
	*/
	fn get_fine_grained(&self) -> LockdownOptions {
		match self {
			Lockdown::Global { enable }	=> {
				if *enable {
					LockdownOptions {
						seccomp_whitelist:	true,
						landlock:		true,
					}
				} else {
					LockdownOptions {
						seccomp_whitelist:	false,
						landlock:		false,
					}
				}
			}
			Lockdown::FineGrained {
				seccomp, landlock
			}				=> {
				LockdownOptions {
					seccomp_whitelist:	*seccomp,
					landlock:		*landlock,
				}
			}
		}
	}
}

impl Default for Lockdown {
	fn default() -> Self {
		Self::Global { enable: false }
	}
}

impl Default for Privacy {
	fn default() -> Self {
		Self {
			lockdown:		Lockdown::default(),
			x11_compat:		false,
			classic_notif:		false,
			push_notification:	true,
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
