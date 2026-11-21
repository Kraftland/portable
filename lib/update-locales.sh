#!/usr/bin/bash

# Generate .po files for Dynamic Permissions

xgettext \
	lib/config/src/definitions/consent.rs \
	--package-name="org.kraftland.portable" \
	--package-version="0.1.0" \
	--copyright-holder="Kraftland Organisation" \
	--language=Rust \
	--from-code=UTF-8 \
	--output=po/messages.pot

for file in $(find po/ -maxdepth 1 -type f); do
	if [[ "${file}" = "po/messages.pot" ]]; then
		continue
	fi

	msgmerge \
		--update \
		--backup=none \
		"${file}" \
		po/messages.pot
done