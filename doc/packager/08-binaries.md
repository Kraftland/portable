# Binaries in a package

A package may contain binaries located under `/usr/bin`, which is intended for a user to start a program without typing a full path in advance. This is a known leaking point described before, and running Packer in either mode will automatically delete the entire directory (albeit replacing them with stub scripts).

While this is a must-have security wise, it can also affect packages whose binaries are expected under `/usr/bin`. To workaround this, a new configuration option: `exec.overlay` may be enabled. Once flipped on, it does the following things:

- Nuke the original `/usr/bin` directory in the sandbox
- Create an overlay filesystem in place
	- The overlay filesystem stacks several sources together. It has many layers of sources, higher layer overwrites lower ones. (e.g. You have both fileA in the top layer and bottom one, the top one will take precedence and appear in `/usr/bin`)
		- Bottom layer: real `/usr/bin`
		- Top layer: `/usr/lib/portable/info/<appID>/bin`

