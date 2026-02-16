# Why Portable?

---

Portable is designed to be simple to use, privacy first and efficient while operating. It has many unique features:

- Network filtering (requires netsock running)
- Background Portal support
- Better integration of your host system
- PipeWire security context
- File bridge: share files to legacy applications without exposing host filesystem
- D-Bus filtering: mandatory filtering with built-in rules to prevent sandbox escape from bad packaging
- Accessibility support: locked down but functional screen reader
- Process Management: process tracking and configurable shutdown behaviour
- Packaging friendly: simple KEY=VAL configuration, with backwards compatibility
- Storage efficient: uses host libraries
- multi-GPU awareness, blocks discrete GPU wakeups by design
- Game Mode for automatic discrete GPU utilisation
- Package defined input device expose, use controllers without exposing the whole `/sys` and `/dev` directory
- Automatic input method workarounds
- Curated, safe list of Portals exposed
- Optional, user defined shared path via environment variable `bwBindPar=/path`

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