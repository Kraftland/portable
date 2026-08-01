#[derive(thiserror::Error, Debug)]
pub enum StartAppError {
	#[error("Could not create proxy to Manager: {0:#?}")]
	ManagerProxyError(zbus::Error),

	#[error("Could not generate properties for Manager: {0:#?}")]
	PropertiesError(zbus::zvariant::Error),
}

#[cfg(feature = "systemd")]
impl crate::spawn::Start for crate::spawn::Spawn {
	async fn start(
		&self,
		dbus_conn:	&zbus::Connection,
	) -> Result<(), crate::spawn::StartAppError> {
		let proxy = zbus_systemd::systemd1::ManagerProxy::new(dbus_conn)
			.await
			.map_err(StartAppError::ManagerProxyError)
			?;

		let unit_name = ServiceName::new(&self.app_id, &self.uid).await;
		let properties = generate_properties(&self.app_id).await?;

		proxy.start_transient_unit(
			unit_name.inner().await.to_string(),
			String::from("replace"),
			properties,

			/*
				The fourth parameter is currently not used and should be
					passed as empty array.
				https://systemd.io/CONTROL_GROUP_INTERFACE/
			*/
			vec![],
		).await;





		Ok(())
	}
}

pub struct ServiceName {
	name:		String,
}

impl ServiceName {
	async fn new(app_id: &str, uid: &str) -> Self {
		let mut unit_name = String::from("app-portable-");
		unit_name.push_str(app_id);
		unit_name.push_str("@");
		unit_name.push_str(uid);
		unit_name.push_str(".service");
		Self { name: unit_name }
	}
	pub async fn inner(&self) -> &str {
		&self.name
	}
}

/**
	Properties are defined in systemd source, scattered around various vtables.

	Notably, those files contain said vtables:

		https://github.com/systemd/systemd/blob/main/src/core/dbus-execute.c

		https://github.com/systemd/systemd/blob/main/src/core/dbus-cgroup.c

		https://github.com/systemd/systemd/blob/main/src/core/dbus-unit.c

		https://github.com/systemd/systemd/blob/main/src/core/dbus-service.c

	I hate their docs. What a genius and brillant move to hide those!
*/
async fn generate_properties(
	app_id:		&str,
) -> Result<Vec<(String, zbus::zvariant::OwnedValue)>, StartAppError> {
	let mut vec: Vec<(String, zbus::zvariant::OwnedValue)> = vec![];

	use zbus::zvariant::{OwnedValue, Str};


	vec.push(
		(
			String::from("Slice"),
			OwnedValue::from(Str::from("app.slice")),
		),
	);

	vec.push(
		(
			String::from("Delegate"),
			OwnedValue::from(true),
		),
	);

	vec.push(
		(
			String::from("DelegateSubgroup"),
			OwnedValue::from(Str::from("portable-cgroup")),
		),
	);

	vec.push(
		(
			String::from("Description"),
			OwnedValue::from(Str::from({
				let mut desc = String::from("Portable sandbox: ");
				desc.push_str(app_id);
				desc
			}))
		),
	);

	vec.push(
		(
			String::from("After"),
			{
				let str_vec = vec![
					Str::from("pipewire.service"),
					Str::from("pipewire-pulse.service"),
					Str::from("xdg-desktop-portal.service"),
				];
				let array = zbus::zvariant::Array::from(str_vec);
				zbus::zvariant::Value::Array(array)
					.try_into()
					.map_err(StartAppError::PropertiesError)
					?
			},
		)
	);

	vec.push(
		(
			String::from("Documentation"),
			{
				let str_vec = vec![
					Str::from("https://github.com/Kraftland/portable"),
				];
				let array = zbus::zvariant::Array::from(str_vec);
				zbus::zvariant::Value::Array(array)
					.try_into()
					.map_err(StartAppError::PropertiesError)
					?
			}
		)
	);

	vec.push(
		(
			String::from("ExitType"),
			OwnedValue::from(Str::from("cgroup")),
		)
	);

	vec.push(
		(
			String::from("SuccessExitStatus"),
			{
				let kill_signals: (Vec<i32>, Vec<i32>) = (vec![9], vec![15]);
				let value = zbus::zvariant::Value::from(kill_signals);
				value
					.try_into()
					.map_err(StartAppError::PropertiesError)
					?
			}
		)
	);

	vec.push(
		(
			String::from("NotifyAccess"),
			OwnedValue::from(Str::from("all")),
		)
	);

	vec.push(
		(
			String::from("NoNewPrivileges"),
			OwnedValue::from(true),
		)
	);

	vec.push(
		(
			String::from("KillMode"),
			OwnedValue::from(Str::from("control-group")),
		)
	);

	vec.push(
		(
			String::from("IPAccounting"),
			OwnedValue::from(true),
		)
	);

	vec.push(
		(
			String::from("MemoryPressureWatch"),
			OwnedValue::from(Str::from("on")),
		)
	);

	vec.push(
		(
			String::from("OOMPolicy"),
			OwnedValue::from(Str::from("kill")),
		)
	);

	vec.push(
		(
			String::from("SyslogIdentifier"),
			OwnedValue::from(Str::from(app_id)),
		)
	);

	vec.push(
		(
			String::from("PrivateIPC"),
			OwnedValue::from(true),
		)
	);

	vec.push(
		(
			String::from("PrivatePIDs"),
			OwnedValue::from(true),
		)
	);

	vec.push(
		(
			String::from("ProtectClock"),
			OwnedValue::from(true),
		)
	);

	// Required for --proc to work
	vec.push(
		(
			String::from("ProtectKernelLogs"),
			OwnedValue::from(false),
		)
	);

	vec.push(
		(
			String::from("RestrictAddressFamilies"),
			{
				let vector = vec![
					"AF_UNIX",
					"AF_INET",
					"AF_INET6",
					"AF_NETLINK",
				];
				let array = zbus::zvariant::Array::from(vector);
				let value = zbus::zvariant::Structure::from((true, array));
				value
					.try_into()
					.map_err(StartAppError::PropertiesError)
					?
			}
		)
	);

	vec.push(
		(
			String::from("CapabilityBoundingSet"),
			/*
				Bitmask here
			*/
			OwnedValue::from(0 as u64),
		)
	);

	vec.push(
		(
			String::from("RestrictSUIDSGID"),
			OwnedValue::from(true),
		)
	);

	vec.push(
		(
			String::from("LockPersonality"),
			OwnedValue::from(true),
		)
	);

	vec.push(
		(
			String::from("RestrictRealtime"),
			OwnedValue::from(true),
		)
	);

	vec.push(
		(
			String::from("ProtectProc"),
			OwnedValue::from(Str::from("invisible")),
			/*
				https://github.com/systemd/systemd/blob/6a863b4dc31adc49fdfdd5deba32ed1b115adda3/src/core/namespace.h#L40
			*/
		)
	);

	vec.push(
		(
			String::from("ProcSubset"),
			Str::from("pid").into(),
		)
	);

	vec.push(
		(
			"PrivateUsers".into(),
			true.into(),
		)
	);

	vec.push(
		(
			"ProtectControlGroups".into(),
			true.into(),
		)
	);
	vec.push(
		(
			"ProtectControlGroupsEx".into(),
			Str::from("private").into(),
		)
	);

	vec.push(
		(
			"PrivateMounts".into(),
			true.into(),
		)
	);

	vec.push(
		(
			"ProtectHome".into(),
			Str::from("no").into(),
		)
	);

	vec.push(
		(
			"KeyringMode".into(),
			Str::from("private").into(),
		)
	);

	vec.push(
		(
			"TimeoutStopUSec".into(),
			(std::time::Duration::from_mins(1).as_micros() as u64).into()
		)
	);

	vec.push(
		(
			"UnsetEnvironment".into(),
			{
				let list = vec![
					"GNOME_SETUP_DISPLAY",
					"GDM_LANG",
					"GDMSESSION",
					"PIPEWIRE_REMOTE",
					"PAM_KWALLET5_LOGIN",
					"GTK2_RC_FILES",
					"ICEAUTHORITY",
					"MANAGERPID",
					"INVOCATION_ID",
					"MANAGERPIDFDID",
					"SSH_AUTH_SOCK",
					"DESKTOP_SESSION",
					"SHELL",
					"__EGL_VENDOR_LIBRARY_FILENAMES",
					"__GLX_VENDOR_LIBRARY_NAME",
					"VK_LOADER_DRIVERS_SELECT",
					"VK_LOADER_DRIVERS_DISABLE",
					"MAIL",
					"SYSTEMD_EXEC_PID",
				];
				let array = zbus::zvariant::Array::from(list);
				zbus::zvariant::Value::from(array)
					.try_into()
					.map_err(StartAppError::PropertiesError)
					?
			}
		)
	);

	vec.push(
		(
			"SystemCallFilter".into(),
			{
				let whitelist = false;
				let deny_list = vec![
					"@clock",
					"@cpu-emulation",
					"@module",
					"@obsolete",
					"@raw-io",
					"@reboot",
					"@swap",
				];
				let array = zbus::zvariant::Array::from(deny_list);

				let array_val = zbus::zvariant::Value::from(array);
				zbus::zvariant::Structure::from(
					(zbus::zvariant::Value::from(whitelist), array_val),
				)
					.try_into()
					.map_err(StartAppError::PropertiesError)
					?
			},
		)
	);

	/*
		TimeoutStartSec was not ported, we have stable systemd notify impl
		SecureBits was not ported. It seems to require value 32 (bit mask 1 << 5)
	*/

	Ok(vec)
}
