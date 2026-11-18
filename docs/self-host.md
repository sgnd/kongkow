# Run the scheduler yourself

Scheduled posts (`kongkow post schedule`) are sent by
`kongkow post status --run`. Run it in a loop on your own computer or
server, so posts go out on time without any cloud service.

## The loop

```bash
kongkow post status --run --every 1m
```

- It sends every due post now, then checks again every minute.
- It prints one line of output for each round in which a post was sent.
- A failed post does not stop it; `kongkow post status` shows why it failed.
- Ctrl-C (or SIGTERM) stops it.
- `--every` is at least 10 seconds.

The machine needs the same `~/.kongkow` as the one where you wrote the
posts. If the vault has a passphrase, the loop needs `KONGKOW_VAULT_PASS`
in its environment; keep that in a file only your user can read.

## macOS: launchd

Save as `~/Library/LaunchAgents/xyz.kongkow.scheduler.plist`, with the path
from `which kongkow`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>xyz.kongkow.scheduler</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/kongkow</string>
    <string>post</string><string>status</string>
    <string>--run</string><string>--every</string><string>1m</string>
    <string>-o</string><string>json</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>/tmp/kongkow-scheduler.log</string>
  <key>StandardErrorPath</key><string>/tmp/kongkow-scheduler.log</string>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/xyz.kongkow.scheduler.plist
```

## Linux: systemd (user service)

Save as `~/.config/systemd/user/kongkow-scheduler.service`:

```ini
[Unit]
Description=kongkow scheduler

[Service]
ExecStart=%h/.local/bin/kongkow post status --run --every 1m -o json
Restart=on-failure
# EnvironmentFile=%h/.config/kongkow/scheduler.env   # KONGKOW_VAULT_PASS, mode 0600

[Install]
WantedBy=default.target
```

```bash
systemctl --user enable --now kongkow-scheduler
loginctl enable-linger "$USER"   # keep it running after you log out
```

## Check it

```bash
kongkow post status
kongkow doctor        # warns about a post stuck running
kongkow audit --limit 10
```
