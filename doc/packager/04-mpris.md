# MPRIS media control

MPRIS is used by applications to expose their playback state.

---

Configuration options:

	- mprisName
		- Default: The last part of the application ID, if left empty
		- Possible values: ASCII characters
		- Description: Defines which client name the sandboxed application shall own.

---

Desktop environments have a feature: Media Controls. It uses D-Bus interface `org.mpris.MediaPlayer2` to communicate.

Warning: enabling the MPRIS feature can be a security risk! Clients may be able to spy on which the user is playing and use it to track people.

# Obtaining client ID

There are a number of ways to obtain D-Bus interface name. We will describe some of them here:

## From Flatpak

Take Spotify as an example, you can invoke `flatpak install spotify` and see there are a number of D-Bus access holes opened up. One of them is for example `org.mpris.MediaPlayer2.spotify`. What you should do is to strip `org.mpris.MediaPlayer2` to `spotify`, and fill it as `mprisName`.

## From D-Bus

### Bare metal

Install `playerctl` first.

Run the application without sandboxing, then run `playerctl -l`. Which should directly return the client name, aka `mprisName`.

### From D-Bus proxy

Invoke portable with environment variable: `PORTABLE_LOGGING=debug`. Then inspect D-Bus session log via command:

```bash
journalctl --user -eu "${friendlyName}-dbus".service
```

# Icon fix

MPRIS daemon can display a blank icon if there isn't a .desktop file to match. This can be fixed by providing a stub .desktop file.

You can use application `D-Spy` to inspect BusName `org.mpris.MediaPlayer2.${mprisName}` (org.mpris.MediaPlayer2.spotify in this case), then look for `DesktopEntry` under _Object Path_ `/org/mpris/MediaPlayer2` > _Interfaces_ > _`org.mpris.MediaPlayer2`_ > _Properties_. Then press _Execute_ to get a reply. For example `(<'spotify'>,)`. Please strip the surrounding characters, to get the .desktop file name.

Then, use the following .desktop file template (Please replace `Name`, `Icon` and `Exec` entry):

```desktop
[Desktop Entry]
Type=Application
Name=AppName
GenericName=Stub for MPRIS
Icon=IconName
TryExec=portable
Exec=env _portableConfig="appID" portable -- %u
Terminal=false
NoDisplay=true
```

And, just install it into /usr/share/applications/, with the name you just obtained, "spotify.desktop" in this case. Example below:

```bash
    echo '''[Desktop Entry]
Type=Application
Name=Spotify Music
GenericName=Stub for MPRIS
Icon=spotify
TryExec=portable
Exec=spotify --uri=%u
Terminal=false
NoDisplay=true''' >"${pkgdir}/usr/share/applications/spotify.desktop"
```