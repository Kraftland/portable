
// Pre runs first, then Post
pub enum FunctionLayer{
	Pre,
	Post,
}

pub struct StopFunc {
	pub layer:	FunctionLayer,
	pub function:	Box<dyn FnOnce() + Send>,
}

pub async fn stop_worker(
	mut rx: tokio::sync::mpsc::Receiver<StopFunc>,
	cancel_token: tokio_util::sync::CancellationToken,
) {
	let mut pre_funcs = vec![];
	let mut post_funcs = vec![];
	let pre_tracker = tokio_util::task::TaskTracker::new();
	let post_tracker = tokio_util::task::TaskTracker::new();

	let mut sigterm = tokio::signal::unix::signal(
		tokio::signal::unix::SignalKind::terminate(),
	).expect("Could not setup SIGTERM listener");

	loop {
		tokio::select! {
			func	=	rx.recv()			=> {
				match func {
					Some(v)	=> {
						match v.layer {
							FunctionLayer::Pre	=> {
								pre_funcs.push(v.function)
							}
							FunctionLayer::Post	=> {
								post_funcs.push(v.function)
							}
						};
					},
					None	=> {break}
				}
			}
			_	=	cancel_token.cancelled()	=> {
				#[cfg(debug_assertions)]
				println!("Shutting down on cancel_token...");

				break;
			}
			_	=	tokio::signal::ctrl_c()		=> {
				#[cfg(debug_assertions)]
				println!("Shutting down on SIGINT...");

				break;
			}
			_	=	sigterm.recv()			=> {

				#[cfg(debug_assertions)]
				println!("Shutting down on SIGTERM...");

				break;
			}
		};
	}

	for func in pre_funcs {
		pre_tracker.spawn(
			async move {
				func()
			},
		);
	};
	
	pre_tracker.close();
	pre_tracker.wait().await;

	for func in post_funcs {
		post_tracker.spawn(
			async move {
				func()
			},
		);
	};
	
	post_tracker.close();
	post_tracker.wait().await;
}
