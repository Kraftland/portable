impl super::PermissionType {
	/**
		Translate the given PermissionType to table name for Portal calls
	*/
	pub fn table(&self) -> &str {
		match self {
			Self::XDPScreenShot	=> {
				"screenshot"
			}
			Self::XDPLocation	=> {
				"location"
			}
			Self::XDPNotifications	=> {
				"notifications"
			}
			Self::ExposeRW		=> {
				"top.kimiblock.Portable"
			}
			Self::ExposeRO		=> {
				"top.kimiblock.Portable"
			}
			Self::ExposeDevice	=> {
				"top.kimiblock.Portable"
			}

			Self::DynamicPermissions(_) => {
				"top.kimiblock.Portable"
			}
		}
	}

	/**
		Translate the given PermissionType to id (objects) for Portal calls
	*/
	pub fn id(&self) -> &str {
		match self {
			Self::XDPScreenShot	=> {
				"screenshot"
			}
			Self::XDPLocation	=> {
				"location"
			}
			Self::XDPNotifications	=> {
				"notification"
			}
			Self::ExposeRW		=> {
				"expose-rw"
			}
			Self::ExposeRO		=> {
				"expose-ro"
			}
			Self::ExposeDevice	=> {
				"expose-device"
			}
			Self::DynamicPermissions(_)
			=> {
				"dynamic-permissions"
			}
		}
	}
}
