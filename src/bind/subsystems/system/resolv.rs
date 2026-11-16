/**
	Probe if /etc/resolv.conf is a symbolic link.

	If true, then we mount the canonicalised destination and create a symbolic link to it.

	If false, we just mount it as-is.

	Note that originally this function did not fail loudly. Rather, it creates warning journal
	that no one would really care. Since it breaks networking it's better to just error out.
*/
pub async fn mount() -> Result<crate::bind::types::BindRules, ResolvError> {
	let attrs = tokio::fs::symlink_metadata("/etc/resolv.conf")
		.await
		.map_err(ResolvError::IOMetaDataError)
		?;

	if attrs.is_symlink() {
		let dest = tokio::fs::canonicalize("/etc/resolv.conf")
			.await
			.map_err(ResolvError::IOCanonicaliseError)
			?;

		/*
			Because of the /etc isolation, we must mount the correct resolv.conf.
		*/
		Ok(
			vec![
				crate::bind::types::BindRule::Path {
					source:	dest.to_path_buf(),
					dest:	dest.to_path_buf(),
					class:	crate::bind::types::BindType::ReadOnly,
				},
				crate::bind::types::BindRule::Symlink {
					source:	dest,
					dest:	"/etc/resolv.conf".into(),
				},
			]
		)
	} else {
		Ok(
			vec![
				crate::bind::types::BindRule::Path {
					source:	"/etc/resolv.conf".into(),
					dest:	"/etc/resolv.conf".into(),
					class:	crate::bind::types::BindType::ReadOnly,
				}
			]
		)
	}
}

#[derive(thiserror::Error, Debug)]
pub enum ResolvError {
	#[error("I/O error reading metadata: {0}")]
	IOMetaDataError(std::io::Error),

	#[error("I/O error performing path canonicalisation: {0}")]
	IOCanonicaliseError(std::io::Error),
}
