use zbus::interface;

pub struct Info;

pub struct Controller {
	pub stop_tx:	tokio::sync::mpsc::Sender<crate::stop::StopLevel>,
}

#[interface (name = "top.kimiblock.Portable.Controller")]
impl Controller {
	#[zbus(name = "Stop")]
	pub async fn stop(&self) {
		self.stop_tx.send(crate::stop::StopLevel::Normal)
			.await
			.expect("Could not send stop signal");
	}
}
