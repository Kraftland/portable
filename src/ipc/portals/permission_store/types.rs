impl super::PermissionType {
	/**
		Translate the given PermissionType to table name for Portal calls
	*/
	pub fn table(&self) -> &str {
		match self {
			Self::XDPScreenShot	=> {
				"screenshot"
			}
			Self::XDPLocation		=> {
				"location"
			}
			Self::XDPNotifications	=> {
				"notifications"
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
			Self::XDPLocation		=> {
				"location"
			}
			Self::XDPNotifications	=> {
				"notification"
			}
		}
	}
}
