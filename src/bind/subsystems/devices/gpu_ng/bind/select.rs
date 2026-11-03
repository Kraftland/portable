/**
	The public function select decides a list of GPUs to expose inside said sandbox.

	It should also convert them into bind rules, including vendor workarounds.

	It accepts a collection of GPU info and perform various decisions, some even be heuristics
		to select a primary GPU (or GPUs).

	If game mode is enabled, then we short circuit to all GPUs.
*/
pub async fn select<T>(
	devices:	T,
	all_gpus:	&bool,
	log_tx:		crate::logger::LogSender,
)
->
	crate::bind::types::BindRules
where
	T:	IntoIterator<Item = super::GPUInfo>,
{
	let gpu_map = if *all_gpus {
		let mut gpu_map = vec![];
		for dev in devices {
			gpu_map.push((dev, true));
		};
		gpu_map
	} else {
		let mut gpu_map = vec![];
		for dev in devices {
			if dev.boot_display {
				gpu_map.push((dev, true));
			} else {
				gpu_map.push((dev, false));
			};
		};
		gpu_map
	};

	/*
		This is split to two stages, early and late.

		Early includes absent hooks only to avoid masking present hooks.

		Late is currently dedicated to present hooks and general udev device binds.
	*/
	let (
		(
			vendor_bind_late_tx,
			mut vendor_bind_late_rx,
		),
		(
			vendor_bind_early_tx,
			mut vendor_bind_early_rx,
		),
	) = {
		(
			tokio::sync::mpsc::unbounded_channel::<crate::bind::types::BindRules>(),
			tokio::sync::mpsc::unbounded_channel::<crate::bind::types::BindRules>(),
		)
	};

	let mut workers: Vec<tokio::task::JoinHandle<Result<(), SelectError>>> = vec![];

	for (device, expose) in gpu_map {
		let (
			vendor_bind_late,
			vendor_bind_early,
		) = {
			(
				vendor_bind_late_tx.clone(),
				vendor_bind_early_tx.clone(),
			)
		};

		workers.push(tokio::spawn(async move {
			use crate::bind::subsystems::devices::gpu_ng::vendor::VendorHook;
			// If we don't expose this device, run absent hooks only
			if ! expose {
				vendor_bind_early.send(
					device
						.vendor
						.absent()
						.await
						.map_err(SelectError::VendorError)
						?
				)
					.map_err(SelectError::TokioSendError)
					?;

				return Ok(());
			};

			let rules: Vec<crate::bind::types::BindRule> = (&device).into();

			vendor_bind_late.send(
				rules,
			)
				.map_err(SelectError::TokioSendError)
				?;

			vendor_bind_late.send(
				device
					.vendor
					.present()
					.await
					.map_err(SelectError::VendorError)
					?
			)
				.map_err(SelectError::TokioSendError)
				?;

			Ok(())
		}));
	};

	{
		for worker in workers {
			match worker.await.map_err(SelectError::TokioJoinError) {
				Ok(v)	=> {
					match v {
						Ok(_)	=> {}
						Err(e)	=> {
							use crate::logger::messages::warn::WarnMessage;
							let _ = log_tx.send(
								crate::logger::messages::LogMessage::Warn(
									WarnMessage::GPUSelectError(e)
								)
							).await;
						}
					}
				}
				Err(e)	=> {
					use crate::logger::messages::warn::WarnMessage;
					let _ = log_tx.send(
						crate::logger::messages::LogMessage::Warn(
							WarnMessage::GPUSelectError(e)
						)
					).await;
				}
			}
		}
	};

	let bind_rules = {
		let mut rules = vec![];

		while let Some(v) = vendor_bind_early_rx.recv().await {
			rules.extend(v);
		}

		while let Some(v) = vendor_bind_late_rx.recv().await {
			rules.extend(v);
		}

		rules
	};

	bind_rules
}

#[derive(thiserror::Error, Debug)]
pub enum SelectError {
	#[error("Could not run the vendor hook: {0}")]
	VendorError(crate::bind::subsystems::devices::gpu_ng::VendorError),

	#[error("Could not send to tokio channel: {0}")]
	TokioSendError(tokio::sync::mpsc::error::SendError<crate::bind::types::BindRules>),

	#[error("Could not spawn GPU work: {0}")]
	TokioJoinError(tokio::task::JoinError),
}
