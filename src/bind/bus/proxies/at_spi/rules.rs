use crate::bind::bus::rules::BusAccessLevel;

pub async fn generate_rules() -> Result<Vec<BusAccessLevel>, super::AtspiError> {
	use crate::bind::bus::rules::BusAccessLevel;
	use crate::bind::bus::rules::BusName;

	let ret = vec![
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.a11y.atspi.Registry")
					.map_err(super::AtspiError::InvalidBusNameError)
					?,
			method: "org.a11y.atspi.Socket.Embed".into(),
			object_path: "/org/a11y/atspi/accessible/root".into(),
		},

		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.a11y.atspi.Registry")
					.map_err(super::AtspiError::InvalidBusNameError)
					?,
			method: "org.a11y.atspi.Socket.Unembed".into(),
			object_path: "/org/a11y/atspi/accessible/root".into(),
		},

		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.a11y.atspi.Registry")
					.map_err(super::AtspiError::InvalidBusNameError)
					?,
			method: "org.a11y.atspi.Registry.GetRegisteredEvents".into(),
			object_path: "/org/a11y/atspi/accessible/root".into(),
		},

		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.a11y.atspi.Registry")
					.map_err(super::AtspiError::InvalidBusNameError)
					?,
			method: "org.a11y.atspi.DeviceEventController.GetKeystrokeListeners".into(),
			object_path: "/org/a11y/atspi/registry/deviceeventcontroller".into(),
		},
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.a11y.atspi.Registry")
					.map_err(super::AtspiError::InvalidBusNameError)
					?,
			method: "org.a11y.atspi.DeviceEventController.GetDeviceEventListeners".into(),
			object_path: "/org/a11y/atspi/registry/deviceeventcontroller".into(),
		},
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.a11y.atspi.Registry")
					.map_err(super::AtspiError::InvalidBusNameError)
					?,
			method: "org.a11y.atspi.DeviceEventController.NotifyListenersSync".into(),
			object_path: "/org/a11y/atspi/registry/deviceeventcontroller".into(),
		},
		BusAccessLevel::Call {
			bus_name: BusName::try_from("org.a11y.atspi.Registry")
					.map_err(super::AtspiError::InvalidBusNameError)
					?,
			method: "org.a11y.atspi.DeviceEventController.NotifyListenersAsync".into(),
			object_path: "/org/a11y/atspi/registry/deviceeventcontroller".into(),
		},
	];

	Ok(ret)
}
