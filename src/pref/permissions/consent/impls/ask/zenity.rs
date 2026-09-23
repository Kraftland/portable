impl crate::pref::permissions::consent::AskConsent for crate::pref::permissions::consent::DynamicPermissions {
	async fn ask(content: Self)
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
