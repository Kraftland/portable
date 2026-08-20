#[derive(Debug)]
pub enum PortalResponse {
	Success,	// 0
	Cancelled,	// 1
	Failed,		// 2
	Unknown {code: u32},
}

pub type Token = String;

/**
	The public function get_response listens on the bus, and returns the underlying
		reply including the success status and a D-Bus Variant.

	Sender is the unique name on Bus,
		and Token is a randomly generated string from generate_token.

	This function returns when listening is ready, and you must listen on a channel to get
		reply.
*/
pub async fn get_response(
	bus:	&zbus::Connection,
	sender:	&str,
	token:	&Token,
) -> zbus::Result<tokio::sync::oneshot::Receiver<zbus::Result<(PortalResponse, zbus::zvariant::OwnedValue)>>> {
	let sender = {
		let name = sender
			.trim_start_matches(":")
			.replace(".", "_");

		name
	};

	let path = {
		let mut pth = String::from("/org/freedesktop/portal/desktop/request/");
		pth.push_str(&sender);
		pth.push_str("/");
		pth.push_str(&token);
		pth
	};

	let proxy = RequestsProxy::new(&bus, path)
		.await
		?;

	let ready_token = tokio_util::sync::CancellationToken::new();

	let (tx, rx) = tokio::sync::oneshot::channel();

	{
		let token = ready_token.clone();
		tokio::spawn(
			async move {
				use futures_util::stream::StreamExt;

				let mut stream = match proxy.receive_response().await {
					Ok(v)	=> {v}
					Err(e)	=> {
						tx.send(
							Err(e)
						);
						return;
					}
				};

				token.cancel();

				let signal = match stream.next().await {
					Some(v)	=> {
						v
					}
					None	=> {
						tx.send(
							Err(zbus::Error::Failure("Empty stream".to_string()))
						);
						return;
					}
				};

				let args: ResponseArgs = match signal.args() {
					Ok(v)	=> v,
					Err(e)	=> {
						tx.send(
							Err(zbus::Error::Failure(format!("{e:#?}")))
						);
						return;
					}
				};

				let status = match args.raw_response {
					0	=> {
						PortalResponse::Success
					}
					1	=> {
						PortalResponse::Cancelled
					}
					2	=> {
						PortalResponse::Failed
					}
					v	=> {
						PortalResponse::Unknown { code: v }
					}
				};

				tx.send(
					Ok(
						(
							status,
							args.value,
						)
					)
				);

				proxy
					.close()
					.await;
			}
		)

	};

	ready_token.cancelled().await;

	Ok(rx)
}

/**
	Generate a token using the rng
*/
pub async fn generate_token() -> Token {
	let mut token = String::from("portable");

	let mut rng = crate::spawn::rng::Rng::new();

	token.push_str(&rng.generate().to_string());

	token
}

#[zbus::proxy(
	interface	= "org.freedesktop.portal.Request",
	default_service	= "org.freedesktop.portal.Desktop",
)]
trait Requests {
	#[zbus(
		signal,
		name	= "Response",
	)]
	async fn response(
		&self,
		raw_response:	u32,
		value:		zbus::zvariant::OwnedValue,
	)	-> zbus::Result<()>;

	async fn close(&self)		-> zbus::Result<()>;
}
