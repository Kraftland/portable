use std::time::Instant;

/**
	Times gettext initialisation
*/
fn main() {
	let start = Instant::now();
	unsafe {
		gettextrs::TextDomain::new("org.kraftland.portable")
			.codeset("UTF-8")
			.init()
			.expect("Could not initialise gettext")
	};

	let elapsed = start.elapsed();
	println!("TextDomain::init(): {:.3} µs", elapsed.as_secs_f64() * 1_000_000.0);
}

