use std::time::{SystemTime, UNIX_EPOCH};

pub struct Rng {
	state: usize,
}

impl Rng {
	/*
		Xorshift64 random number generator
	*/
	pub fn new() -> Self {
		let seed = SystemTime::now()
			.duration_since(UNIX_EPOCH)
			.map(|d| d.as_nanos() as usize)
			.unwrap_or(1225);

		Self {
			state:	seed,
		}
	}

	pub fn generate(&mut self) -> u32 {
		let mut x = self.state;
		x ^= x << 13;
		x ^= x >> 7;
		x ^= x << 17;
		self.state = x;
		let min = 10240;
		let max = 2147483640;
		let res = min + (x % (max - min));
		res as u32
	}
}
