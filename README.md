# Portable
A sandboxing framework, originally designed for WeChat. Still in heavy development.

![Example](https://raw.githubusercontent.com/Kraftland/portable/refs/heads/master/example.webp)

---

# File installment

## Portable

Install aur/portable-git or

```
install -Dm755 portable.sh /usr/bin/portable
install -Dm755 open.sh /usr/lib/portable/open
install -Dm755 user-dirs.dirs /usr/lib/portable/user-dirs.dirs
install -Dm755 mimeapps.list /usr/lib/portable/mimeapps.list
install -Dm755 flatpak-info /usr/lib/portable/flatpak-info
```

## Configurations


Preferred location:

```
# Modify before installing
install -Dm755 config /usr/lib/portable/info/appID/config
```

## Runtime

Environment variables are read from `XDG_DATA_HOME/stateDirectory/portable.env`

Start portable with environment variable `_portalConfig`, which is pointed to the actual config.

### Debugging

Start portable with argument `--actions connect-tty debug-shell`
