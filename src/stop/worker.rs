use super::get::StopMessage;

pub async fn stop(
	mut rx:	tokio::sync::mpsc::UnboundedReceiver<StopMessage>,
	pre:	tokio_util::sync::CancellationToken,
	post:	tokio_util::sync::CancellationToken,
	succ:	bool,
) {
	rx.close();

	let (pre_tasks, post_tasks) = {
		let mut pre = vec![];
		let mut post = vec![];

		while let Some(v) = rx.recv().await {
			match v {
				super::get::StopMessage::Post { task }		=> {
					post.push(task);
				}
				super::get::StopMessage::Prepare { task }	=> {
					pre.push(task);
				}
			}
		};
		(pre, post)
	};

	pre.cancel();

	for pre in pre_tasks {
		let res = match pre.await {
			Ok(v)	=> {v}
			Err(e)	=> {
				eprintln!("Could not spawn stop task: {e:#?}");
				continue;
			}
		};
		match res {
			Ok(_)	=> {}
			Err(e)	=> {
				eprintln!("Could not execute stop task: {e:#?}")
			}
		};
	};

	post.cancel();

	for post in post_tasks {
		let res = match post.await {
			Ok(v)	=> {v}
			Err(e)	=> {
				eprintln!("Could not spawn stop task: {e:#?}");
				continue;
			}
		};
		match res {
			Ok(_)	=> {}
			Err(e)	=> {
				eprintln!("Could not execute stop task: {e:#?}")
			}
		};
	};

	#[cfg(debug_assertions)]
	println!("Finished stop sequence");

	if succ {
		std::process::exit(0)
	} else {
		std::process::exit(1)
	}
}
