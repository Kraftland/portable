### CLI arguments

Useful to include in .desktop entries as actions

---

Environment variables:

- PORTABLE_LOGGING	-> Optional
	- Possible values: debug, info

- _portableConfig	-> Required
	- Possible values:
		- Application ID of installed sandbox under /usr/lib/portable/info (recommended), or XDG_CONFIG_DIR/portable/info
			- The configuration will be read from info/ID/config
		- Relative or absolute path to a configuration file


Command line arguments (optional):

```
	--actions <action>
		debug-shell	-> Enter the sandbox via a bash shell
		opendir	-	-> Open the sandbox's home directory
		share-files	-> Place files in sandbox's "Shared" directory
		reset-documents	-> Revoke granted file access permissions
		stats	-	-> Show basic status of the sandbox (if running)
		quit	-	-> Terminate running sandbox
	--	-	-	-> Any argument after this double dash will be passed to the application
	--expose <orig> <dest>	-> See further doc below
```


# Exposing files
The `--expose` flag bind host origin path to sandbox destination. Prefix `<dest>` with ro: to bind read-only, or dev: to bind device. This will not work if the sandbox has already started, but a special mechanism works this around:

Portable opens the host file and pass it into the sandbox as FDs. They will appear under `$XDG_RUNTIME_DIR/doc/<random string>/` with their original name. Portable automatically rewrites the application command line to use those paths, so text editors and other applications can operate smoothly.