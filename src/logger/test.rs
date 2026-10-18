// use super::*;

/**
	Tests that the next generation logger is not bigger in size than older logger.

	It was 32 bytes.
*/
#[test]
#[cfg(not(debug_assertions))]
fn logger_is_not_bigger() {
	use crate::logger::messages;

	println!(
		"Size of Fatal: {}",
		std::mem::size_of::<messages::fatal::FatalMessage>(),
	);

	println!(
		"Size of Info: {}",
		std::mem::size_of::<messages::info::InfoMessage>(),
	);

	println!(
		"Size of Warn: {}",
		std::mem::size_of::<messages::warn::WarnMessage>(),
	);

	let new_size = std::mem::size_of::<messages::LogMessage>();

	println!("Size of next generation logger is {new_size}");

	assert!(
		new_size <= 32
	)
}
