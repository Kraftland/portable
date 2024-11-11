# Portable
A sandboxing framework, originally designed for WeChat. Still in heavy development.

# File installment

## Portable

```
install -Dm755 portable.sh /usr/bin/portable
install -Dm755 open.sh /usr/lib/portable/open
install -Dm755 user-dirs.dirs /usr/lib/portable/user-dirs.dirs
install -Dm755 mimeapps.list /usr/lib/portable/mimeapps.list
install -Dm755 flatpak-info /usr/lib/portable/flatpak-info
```

## Configurations

```
# Modify before installing
install -Dm755 config /usr/lib/portable/info/appID/config
```

## Runtime

Environment variables are read from stateDirectory/portable.env
