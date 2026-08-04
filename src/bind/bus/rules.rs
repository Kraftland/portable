/**
	The BusAccessLevel enum is used to define a rule on which sandboxed application is allowed
	to communicate with outside applications.

	It is only for one bus name, thus the final rule would be vector of them.
*/
#[derive(Debug)]
pub enum BusAccessLevel {
	/**
		The name/ID is visible in the ListNames, ListActivatableNames etc.'s reply
		The name's info, such as PID can be retrived
	*/
	See {
		bus_name:	BusName,
	},

	/**
		Allow the sandboxed app to take ownership of said bus name.
		Very dangerous as it allows app to impersonate other services.
	*/
	OwnName {
		bus_name:	BusName,
	},

	/**
		The second most dangerous and open access type.
		It allows the sandboxed application to talk with an outside service unfiltered.

		Internally maps to xdg-dbus-proxy's TALK policy to prevent acquiring names
	*/
	WellknownName {
		bus_name:	BusName,
	},

	/**
		Allow a sandboxed process to call certain methods on certain object paths
	*/
	Call {
		bus_name:	BusName,
		/**
			"method" may be quite misleading, but it actually maps to

				```
				interface name [.] method name
				```

			A .* suffix may be allowed.
		*/
		method:		String,

		/**
			possible with a `/ *` suffix
		*/
		object_path:	String,
	},

	/**
		Allows a sandboxed process to receive broadcasts from outside
	*/
	GetBroadcast {
		bus_name:	BusName,
		/**
			"method" may be quite misleading, but it actually maps to

				```
				interface name [.] method name
				```

			A .* suffix may be allowed.
		*/
		method:		String,

		/**
			possible with a `/ *` suffix
		*/
		object_path:	String,
	},
}

use crate::bind::types::ToCmdline;

impl BusAccessLevel {
	async fn to_cmdline(self)	-> String {
		match self {
			BusAccessLevel::See { bus_name }	=> {
				let mut cmdline = String::from("--see=");
				cmdline.push_str(&bus_name.to_string());
				cmdline
			}
			BusAccessLevel::Call { bus_name, method, object_path }
								=> {
				let mut cmdline = String::from("--call=");
				cmdline.push_str(&bus_name.to_string());
				cmdline.push_str("=");
				cmdline.push_str(&method);
				cmdline.push_str("@");
				cmdline.push_str(&object_path);
				cmdline
			}
			BusAccessLevel::OwnName { bus_name }	=> {
				let mut cmdline = String::from("--own=");
				cmdline.push_str(&bus_name.to_string());
				cmdline
			}
			BusAccessLevel::GetBroadcast { bus_name, method, object_path }
								=> {
				let mut cmdline = String::from("--broadcast=");
				cmdline.push_str(&bus_name.to_string());
				cmdline.push_str("=");
				cmdline.push_str(&method);
				cmdline.push_str("@");
				cmdline.push_str(&object_path);
				cmdline
			}
			BusAccessLevel::WellknownName { bus_name }
								=> {
				let mut cmdline = String::from("--talk=");
				cmdline.push_str(&bus_name.to_string());
				cmdline
			}
		}
	}
}



/**
	The BusName struct is mostly a type alias of String, but wrapped inside a struct to perform
	checks.
*/
#[derive(Debug)]
pub struct BusName {
	name:	String
}

#[derive(thiserror::Error, Debug)]
pub enum BusNameError {
	#[error("Could not convert type to D-Bus name: bus name must not exceed 255 characters (0)")]
	BusNameTooLongError(usize),
}

impl Into<String> for BusName {
	fn into(self) -> String {
		self.name
	}
}

impl ToString for BusName {
	fn to_string(&self) -> String {
		self.name.to_owned()
	}
}

impl TryFrom<String> for BusName {
	fn try_from(value: String) -> Result<Self, Self::Error> {
		if value.len() >= 255 {
			Err(BusNameError::BusNameTooLongError(value.len()))
		} else {
			Ok(Self {
				name: value
			})
		}
	}
	type Error = BusNameError;
}

impl TryFrom<&str> for BusName {
	fn try_from(value: &str) -> Result<Self, Self::Error> {
		if value.len() >= 255 {
			Err(BusNameError::BusNameTooLongError(value.len()))
		} else {
			Ok(Self {
				name: value.into()
			})
		}
	}
	type Error = BusNameError;
}
