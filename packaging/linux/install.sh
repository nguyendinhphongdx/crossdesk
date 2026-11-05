#!/bin/sh
# Installs Share Device system-wide. Usage: sudo ./install.sh
set -e
if [ "$(id -u)" -ne 0 ]; then
  exec sudo "$0" "$@"
fi
cd "$(dirname "$0")"
install -m 0755 share-device /usr/local/bin/share-device
install -m 0755 share-device-setup /usr/local/bin/share-device-setup
install -d /usr/local/share/share-device
install -m 0644 70-share-device.rules /usr/local/share/share-device/70-share-device.rules
install -m 0644 share-device.desktop /usr/share/applications/share-device.desktop
/usr/local/bin/share-device-setup
echo "Installed. Open 'Share Device' from your application menu (after logging back in)."
