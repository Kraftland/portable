# Packer

A tool for packagers.

---

```
This is Portable packer, a tool to build sandboxed package
Visit https://github.com/Kraftland/portable for documentation and information.
Supported arguments:
	-v	-	-	-> Verbose output
	--distro [distro name]	-> Specify the distribution.
	--hash [true / false]	-> Enables hashing of configuration file. Currently has no effect. (optional)
	--config [path]	-	-> Specify the configuration source for sandbox
	--dbus-activation	-> Enables the activation from D-Bus (optional)
Exit codes:
	1	-	-	-> Syntax / argument error
	200	-	-	-> Configuration error
```