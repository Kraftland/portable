
// Pre runs first, then Post
pub enum FunctionLayer{
	Pre,
	Post,
}

pub struct StopFunc {
	pub layer:	FunctionLayer,
	pub function:	Box<dyn FnOnce() + Send>,
}

pub enum StopLevel {
	Error (isize),
	Normal,
}

pub async fn stop_worker(
	mut rx: tokio::sync::mpsc::Receiver<StopFunc>,
	mut stop_signal: tokio::sync::mpsc::Receiver<StopLevel>,
) {
	let mut pre_funcs = vec![];
	let mut post_funcs = vec![];


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
			sig	=	stop_signal.recv()	=> {
				#[cfg(debug_assertions)]
				println!("Shutting down on cancel_token...");

				let error = {
					match sig {
						Some(StopLevel::Error(_))	=> {true}
						_				=> {false}
					}
				};

				shutdown(pre_funcs, post_funcs, error).await;
				break;
				// Some(sig)
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
}

async fn shutdown(
	pre_funcs: Vec<Box<dyn FnOnce() + Send>>,
	post_funcs: Vec<Box<dyn FnOnce() + Send>>,
	error_code: bool,
) {
	let pre_tracker = tokio_util::task::TaskTracker::new();
	let post_tracker = tokio_util::task::TaskTracker::new();
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
	if error_code {
		std::process::exit(1);
	}
}
