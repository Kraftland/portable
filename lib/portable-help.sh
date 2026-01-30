#!/bin/bash

# Shellcheck configurations below
# shellcheck disable=SC1090,SC2174,SC2154,SC2129,SC1091,SC2086
# Shellcheck configurations end

function printHelp() {
	echo "This is Portable, a fast, private and efficient Linux desktop sandbox."
	echo "Visit https://github.com/Kraftland/portable for documentation."
	echo -e "\n"
	echo "Environment variables:"
	echo "	PORTABLE_LOGGING	-> Optional"
	echo "		Possible values: debug, info"
	echo "	_portalConfig		-> Required"
	echo "		Possible values: "
	echo "			Application ID of installed sandbox under /usr/lib/portable/info"
	echo "			Relative or absolute path to a configuration file"
	echo -e "\n"
	echo "Command line arguments (optional):"
	echo "	-v	-	-	-> Verbose output"
	echo "	--actions <action>"
	echo "		debug-shell	-> Enter the sandbox via a bash shell"
	echo "		opendir	-	-> Open the sandbox's home directory"
	echo "		share-files	-> Place files in sandbox's \"Shared\" directory"
	echo "		reset-documents	-> Revoke granted file access permissions"
	echo "		stats	-	-> Show basic status of the sandbox (if running)"
	echo "	--	-	-	-> Any argument after this double dash will be passed to the application"
	echo "	--help	-	-	-> Print this help"
	exit 0
}

printHelp