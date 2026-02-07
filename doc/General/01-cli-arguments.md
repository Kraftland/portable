### CLI arguments

Useful to include in .desktop entries

---

Environment variables:

- PORTABLE_LOGGING	-> Optional
	- Possible values: debug, info

- _portalConfig		-> Required
	- Possible values:
		- Application ID of installed sandbox under /usr/lib/portable/info
		- Relative or absolute path to a configuration file


Command line arguments (optional):

```
	-v	-	-	-> Verbose output
	--actions <action>
		debug-shell	-> Enter the sandbox via a bash shell
		opendir	-	-> Open the sandbox's home directory
		share-files	-> Place files in sandbox's "Shared" directory
		reset-documents	-> Revoke granted file access permissions
		stats	-	-> Show basic status of the sandbox (if running)
		quit	-	-> Terminate running sandbox
	--	-	-	-> Any argument after this double dash will be passed to the application
	--help	-	-	-> Print this help
```