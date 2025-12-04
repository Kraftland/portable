# Packer

A tool for packagers.

---

This is Portable packer, a tool to build sandboxed package

```
Supported arguments:
	-v	-	-	-> Verbose output
	--distro [distro name]	-> Specify the distribution.
	--mode [copy] [pkg]	-> Modes of operation
	--hash [true / false]	-> Enables hashing of configuration file. Currently has no effect. (optional)
	--config [path]	-	-> Specify the configuration source for sandbox
	--dbus-activation	-> Enables the activation from D-Bus (optional)
Exit codes:
	1	-	-	-> Syntax / argument error
	200	-	-	-> Configuration error
```

You MUST the .desktop file with modifications, and the configuration file.

# Modes of operation

## Copy from existing package

In this mode, all files will first be copied into the new package. The problematic files will then be automatically removed.

- Arch Linux
	- `--mode copy [pkg name]`