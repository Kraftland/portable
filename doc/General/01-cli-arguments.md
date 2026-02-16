### CLI arguments

Useful to include in .desktop entries as actions

---

Environment variables:

- PORTABLE_LOGGING	-> Optional
	- Possible values: debug, info

- _portableConfig	-> Required
	- Possible values:
		- Application ID of installed sandbox under /usr/lib/portable/info (recommended)
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
```