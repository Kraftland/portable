use thiserror::Error;

#[derive(Debug, Error)]
pub enum SecurityContextError {
	#[error("Error connecting to Wayland socket: {0:#?}")]
	ConnectError(wayrs_client::ConnectError),

	#[error("Error setting blocking roundtrip mode for connection: {0:#?}")]
	SetBlockingRoundtripErr(std::io::Error),

	#[error("Error converting protocol name to C string: {0:#?}")]
	CStringError(std::ffi::NulError),

	#[error("Error binding singleton: {0:#?}")]
	BindManagerErr(wayrs_client::global::BindError),

	#[error("Error creating close_fd: {0:#?}")]
	CloseFdError(std::io::Error),

	#[error("Error creating listen_fd: {0:#?}")]
	ListenFdError(std::io::Error),

	#[error("I/O error: {0:#?}")]
	IOError(std::io::Error),
}

/**
	Create a security context protocol

	Returns the original PathBuf when unsupported
*/
pub async fn create_context(
	original_socket:	std::path::PathBuf,
	portable_runtime:	std::sync::Arc<crate::bind::subsystems::dirs::portable_runtime::PortableRuntime>,
	logger:			crate::logger::LogSender,

	app_id:			String,
	instance_id:		String,
) -> Result<std::path::PathBuf, SecurityContextError> {
	use crate::bind::subsystems::dirs::RuntimePathsTrait;

	let conn = connect_socket()
		.await
		?;

	let supports_security_context = {
		let protocol = std::ffi::CString::new("wp_security_context_manager_v1")
			.map_err(SecurityContextError::CStringError)
			?;
		let globals = conn.globals();

		let mut security_context: bool = false;

		for global in globals {
			if global.interface == protocol {
				security_context = true
			} else {
				continue;
			}
		}

		security_context
	};

	if ! supports_security_context {
		let _ = logger.send(
			crate::logger::LogMessage {
				level: crate::logger::LogLevel::Warn,
				message: format!("Compositor does not support security-context-v1"),
			},
		).await;
		return Ok(original_socket);
	};

	let security_context_path = {
		let mut path = portable_runtime.path();
		path.push("wayland");
		path
	};

	let context_fd: std::os::fd::OwnedFd = {
		let file = tokio::fs::OpenOptions::new()
			.write(true)
			.create_new(true)
			.mode(0o700)
			.open(&security_context_path)
			.await.map_err(SecurityContextError::ListenFdError)
			?;
		file.into_std().await.into()
	};

	listen_context(context_fd, conn, app_id, instance_id).await?;


	Ok(security_context_path)
}

async fn listen_context(
	/*
		listen_fd must be ready to accept new connections
			when this request is sent by the client.
		In other words, the client must call bind(2) and listen(2) before sending the FD.
	*/
	listen_fd:	std::os::fd::OwnedFd,

	mut wayland_conn:	wayrs_client::Connection<()>,

	app_id:		String,

	instance_id:	String,

) -> Result<(), SecurityContextError> {
	let ctx_manager =
		wayland_conn.bind_singleton
		::<wayrs_protocols::security_context_v1::wp_security_context_manager_v1::WpSecurityContextManagerV1> (1)
		.map_err(SecurityContextError::BindManagerErr)
		?;

	/*
		close_fd is a FD that will signal hangup when the compositor should
			stop accepting new connections on listen_fd.
	*/
	let (close_fd_hold, close_fd_compositor) = {
		std::os::unix::net::UnixStream::pair()
			.map_err(SecurityContextError::CloseFdError)
			?
	};

	let listener = ctx_manager
		.create_listener(
			&mut wayland_conn,
			listen_fd,
			close_fd_compositor.into(),
		);

	let sandbox_engine_id = std::ffi::CString::new("top.kimiblock.portable")
		.map_err(SecurityContextError::CStringError)
		?;
	let app_id = std::ffi::CString::new(app_id)
		.map_err(SecurityContextError::CStringError)
		?;
	let instance_id = std::ffi::CString::new(instance_id)
		.map_err(SecurityContextError::CStringError)
		?;

	listener.set_sandbox_engine(
		&mut wayland_conn,
		sandbox_engine_id,
	);
	listener.set_app_id(
		&mut wayland_conn,
		app_id,
	);
	listener.set_instance_id(
		&mut wayland_conn,
		instance_id,
	);
	listener.commit(
		&mut wayland_conn,
	);

	wayland_conn
		.async_flush()
		.await
		.map_err(SecurityContextError::IOError)
		?;

	wayland_conn
		.blocking_roundtrip()
		.map_err(SecurityContextError::SetBlockingRoundtripErr)
		?;

	// Hold the fd until sandbox exits
	tokio::spawn(async move {
		let _fd = close_fd_hold;
		let mut conn = wayland_conn;
		while conn.async_recv_events().await.is_ok() {

		}
	});

	Ok(())
}

/**
	Connect to the Wayland server and set blocking roundtrip mode for socket

	The latter seems to fix flaky sockets.
*/
async fn connect_socket() -> Result<wayrs_client::Connection<()>, SecurityContextError> {
	let mut conn = wayrs_client::Connection::connect()
		.map_err(SecurityContextError::ConnectError)
		?;
	conn
		.blocking_roundtrip()
		.map_err(SecurityContextError::SetBlockingRoundtripErr)
		?;
	Ok(conn)
}
