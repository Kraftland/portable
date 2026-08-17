/**
	The new stop system works differently than that of the previous ones.

	Instead of relying on others to signal exit, this version automatically executes on main()
		after run() has finished.

	It still has two layers of execution, pre and post.

	Instead of issuing functions and send them to the channel, we give cancel tokens away, and
		other parts of the system give in their spawned tokio task that we can await upon.

	When the run() function finishes, a worker is started to start execution.
*/

pub struct Stop {
	pre_cancel:	tokio_util::sync::CancellationToken,
	post_cancel:	tokio_util::sync::CancellationToken,

	stop_funcs:	tokio::sync::Mutex<Vec<StopMessage>>,
}

/**
	Prepare runs first, while Post runs later
*/
pub enum StopMessage {
	Prepare {
		task:	tokio::task::JoinHandle<Result<(), StopError>>
	},
	Post {
		task:	tokio::task::JoinHandle<Result<(), StopError>>
	}
}

#[derive(Debug, thiserror::Error)]
pub enum StopError {
}

impl Stop {
	/**
		new() creates a new instance of Stop struct

		It also sets up a background thread to handle incoming requests.
	*/
	pub async fn new() -> std::sync::Arc<Self> {
		std::sync::Arc::new(
			Self {
				pre_cancel:	tokio_util::sync::CancellationToken::new(),
				post_cancel:	tokio_util::sync::CancellationToken::new(),
				stop_funcs:	tokio::sync::Mutex::new(vec![])
			}
		)
	}

	/**
		Add a task to the pre pool
	*/
	pub async fn add_pre(self, task: tokio::task::JoinHandle<Result<(), StopError>>) {
		let mut inner = self
			.stop_funcs
			.lock()
			.await;

		inner.push(
			StopMessage::Prepare { task: task }
		);
	}

	pub async fn add_post(self, task: tokio::task::JoinHandle<Result<(), StopError>>) {
		let mut inner = self
			.stop_funcs
			.lock()
			.await;

		inner.push(
			StopMessage::Post { task: task }
		);
	}
}
