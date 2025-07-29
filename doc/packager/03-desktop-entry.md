# The .desktop entry

- The file name of your .desktop file **must** match the appID, like `top.kimiblock.example.desktop`
- `TryExec=portable` is recommended
- Your .desktop file *should* contain the following entries:

```
X-Flatpak-Tags=aTag;
X-Flatpak=appID;
X-Flatpak-RenamedFrom=previousName.desktop;
```
