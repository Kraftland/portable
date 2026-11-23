pub fn help_content() -> String {
	format!(
		"{}\n{}\n{}",
		gettextrs::gettext(
			"This is Portable Sandbox. A fast, private, modern sandbox designed for desktop Linux.",
		),
		gettextrs::gettext(
			"Visit https://github.com/Kraftland/portable/blob/master/doc/General/01-cli-arguments.md for CLI documentation"
		),
		gettextrs::gettext(
			"You should also receive a copy from the distributor, usually in /usr/share/doc/portable"
		)
	)
}
