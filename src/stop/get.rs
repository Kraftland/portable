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
	pub pre_parent:		tokio_util::sync::CancellationToken,
	pub post_cancel:	tokio_util::sync::CancellationToken,

	pub stop_funcs:		tokio::sync::mpsc::UnboundedSender<StopMessage>,
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
	#[error("Could not send stop_func: {0:#?}")]
	SendError(tokio::sync::mpsc::error::SendError<StopMessage>),

	#[error("Could not restore console: {0:#?}")]
	RestoreConsoleError(nix::Error),

	#[error("Could not remove directory: {0:#?}")]
	RemoveFsError(std::io::Error),
}

impl Stop {
	/**
		new() creates a new instance of Stop struct
	*/
	pub async fn new() -> (std::sync::Arc<Self>, tokio::sync::mpsc::UnboundedReceiver<StopMessage>) {
		let (tx, rx) = tokio::sync::mpsc::unbounded_channel();

		(
			std::sync::Arc::new(
				Self {
					pre_parent:	tokio_util::sync::CancellationToken::new(),
					post_cancel:	tokio_util::sync::CancellationToken::new(),
					stop_funcs:	tx,
				}
			),
			rx
		)
	}

	/**
		Add a task to the pre pool
	*/
	pub async fn add_pre(self, task: tokio::task::JoinHandle<Result<(), StopError>>)
		-> Result<(), StopError> {
		self
			.stop_funcs
			.clone()
			.send(
				StopMessage::Prepare {
					task: task,
				},
			)
			.map_err(StopError::SendError)
	}

	pub async fn add_post(self, task: tokio::task::JoinHandle<Result<(), StopError>>)
		-> Result<(), StopError> {
		self
			.stop_funcs
			.clone()
			.send(
				StopMessage::Post {
					task: task,
				},
			)
			.map_err(StopError::SendError)
	}
}
