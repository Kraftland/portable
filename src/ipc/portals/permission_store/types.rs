
impl super::PermissionType {
	/**
		Translate the given PermissionType to table name for Portal calls
	*/
	fn table(&self) -> &str {
		match self {
			Self::ScreenShot	=> {
				"screenshot"
			}
			Self::Location		=> {
				"location"
			}
		}
	}

	/**
		Translate the given PermissionType to id (objects) for Portal calls
	*/
	fn id(&self) -> &str {
		match self {
			Self::ScreenShot	=> {
				"screenshot"
			}
			Self::Location		=> {
				"location"
			}
		}
	}
}
