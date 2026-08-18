use zbus::interface;

pub struct Info;

pub struct Controller {
	pub cancel_token:	tokio_util::sync::CancellationToken,
}

#[interface (name = "top.kimiblock.Portable.Controller")]
impl Controller {
	#[zbus(name = "Stop")]
	pub async fn stop(&self) {
		self.cancel_token.cancel();
	}
}
