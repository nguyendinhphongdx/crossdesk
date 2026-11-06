#!/bin/sh
# Installs Crossdesk system-wide. Usage: sudo ./install.sh
set -e
if [ "$(id -u)" -ne 0 ]; then
  exec sudo "$0" "$@"
fi
cd "$(dirname "$0")"
install -m 0755 crossdesk /usr/local/bin/crossdesk
install -m 0755 crossdesk-setup /usr/local/bin/crossdesk-setup
install -d /usr/local/share/crossdesk
install -m 0644 70-crossdesk.rules /usr/local/share/crossdesk/70-crossdesk.rules
install -m 0644 crossdesk.desktop /usr/share/applications/crossdesk.desktop
install -D -m 0644 crossdesk.png /usr/share/icons/hicolor/256x256/apps/crossdesk.png
gtk-update-icon-cache -q /usr/share/icons/hicolor 2>/dev/null || true
/usr/local/bin/crossdesk-setup
echo "Installed. Open 'Crossdesk' from your application menu (after logging back in)."
