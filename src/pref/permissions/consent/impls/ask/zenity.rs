impl crate::pref::permissions::consent::AskConsent for crate::pref::permissions::consent::DynamicPermissions {
	async fn ask(content: Self, config: std::sync::Arc<crate::config::Config>)
		-> Result<Self, Self::ConsentError>
	{
		unimplemented!()
	}

	type ConsentError = ZenityError;
}

#[derive(Debug, thiserror::Error)]
pub enum ZenityError {

}

/**
	A Zenity permission object we implement to translate:

		DynamicPermission (singular) <-> ZenityPermissionObject <-> Vec<String>

	We expect our Zenity dialogue to have 3 columns (albeit 1 hidden):
	Allow (checkbox) | Internal Permission Unique ID (hidden) | Description
*/
struct ZenityPermissionObject {
	default_allow:	bool,
	perm_uid:	String,
	desc:		String,
}

impl From<&portable_config::definitions::consent::DynamicPermission> for ZenityPermissionObject {
	fn from(value: &portable_config::definitions::consent::DynamicPermission) -> Self {
		Self {
			default_allow:	value.default_allow(),
			perm_uid:	value.id().to_string(),
			desc:		format!("{value}"),
		}
	}
}

impl crate::bind::types::ToCmdline for ZenityPermissionObject {
	#[inline]
	async fn to_cmdline(&self)	-> Vec<String> {
		let default_allow = match self.default_allow {
			true	=> {"TRUE"}
			false	=> {"FALSE"}
		};

		vec![
			default_allow.to_string(),
			self.perm_uid.to_string(),
			self.desc.to_string(),
		]
	}
}

/**
	Actual logic here
*/
async fn ask_zenity(
	permissions:	crate::pref::permissions::consent::DynamicPermissions,
	config:		std::sync::Arc<crate::config::Config>,
)
-> Result<crate::pref::permissions::consent::DynamicPermissions, ZenityError> {
	let permission_objects: Vec<ZenityPermissionObject> = {
		let mut vec: Vec<ZenityPermissionObject> = vec![];

		for perm in permissions {
			vec.push(
				(&perm).into()
			);
		};

		vec
	};


	let mut cmdline: Vec<String> = vec![
		String::from("zenity"),
		String::from("--list"),
		String::from("--checkbox"),
		String::from("--multiple"),

		String::from("--title"),
		String::from(config.metadata.display_name),

		String::from("--text=Would like to access..."),

		String::from("--column=Allow"),
		String::from("--column=Identifier"),
		String::from("--column=Permission"),

		String::from("--print-column"),
		String::from("2"),

		String::from("--hide-column"),
		String::from("2"),
	];

	{
		use crate::bind::types::ToCmdline;
		for object in permission_objects {
			cmdline.extend(object.to_cmdline().await);
		}
	};
}
