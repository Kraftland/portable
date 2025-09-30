# Why Portable?

---

Portable is designed to be simple to use, privacy first and efficient while operating. It has many unique features:

- Background Portal support.
- Exposes some theming and preference configurations by default for user-level and all configuration in system-level.
- Wayland Security Context support. (although deprecated, you can use it)
- PipeWire security context when absolutely needed.
- Access Control: Limits what the application can see, write and modify. Sandboxed applications are self-contained.
- Sharing files with the application, even if it doesn't support portals. portable creates a directory within the sandbox home to contain shared files.
- D-Bus filtering & accessibility support: Cuts off unneeded D-Bus messages thus eliminates the possibility to locate, spawn a process outside of the sandbox, mess with the host system and other possible exploits.
- Process Management: Monitors running processes and quit them with one click.
- Packaging Friendly as portable only requires a config file to function.
- Storage efficient compared to Flatpak: Using host system as the "runtime".
- Hybrid GPU workarounds are automatically applied to prevent waking up discrete GPUs, often caused by Vulkan and Electron applications.
- Input Method automatic detection.
- Curated, safe list of Portals exposed. Unsafe Portals will be blocked.

You may want a comparison between Flatpak and Portable:

| Portable | Flatpak |
| ------- | ------------------ |
| :x: | Includes package management |
| :x: | Large package base |
| Efficient on disk while integrates better | Could not use system libraries & resources |
| Input Method workarounds | :x: |
| Hybrid GPU workarounds | :x: |
| File sharing workarounds | :x: |

That said, Flatpak's XDG Desktop Portal is one of our security foundation. So a respect towards that should be given. And another thing to notice here is that we don't act as a source to distribute software, sandboxing is all we do. The distribution package manager takes care of tracking and installing files.

# Types of applications that suits Portable

In theory most applications can run inside Portable, but not all of them worth sandboxing. Here are a list of application characteristics that we thought need Portable.

- The software is NOT open source
- The software has major possibilities of being exploited
- The software is Web Browser based
- The software is a game
- The software wakes up discrete GPU
- The software doesn't follow the _XDG Base Directory_ specifications