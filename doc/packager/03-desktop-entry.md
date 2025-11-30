# The .desktop entry

- The file name of your .desktop file **must** match the appID, like `top.kimiblock.example.desktop`
- `TryExec=portable` is recommended
- For correct window grouping behaviour, please match the last part of the ID to application's own app_id, you can find this out in e.g. *KWin Debug Console*.
- Your .desktop file *should* **NOT** contain the following entries:

```
X-Flatpak-Tags=aTag;
X-Flatpak=appID;
```

Notice: If your app is designated as a single-window program, add `SingleMainWindow=true` to the .desktop file.