#!/bin/bash
set -euo pipefail

echo "=== OpenClaw Relay - Oracle Cloud Setup ==="

# Create user
sudo useradd -r -s /bin/false openclaw 2>/dev/null || true

# Create directories
sudo mkdir -p /var/lib/openclaw-relay/certs
sudo chown -R openclaw:openclaw /var/lib/openclaw-relay

# Allow binding to port 443
sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/openclaw-relay

# Install systemd service
sudo cp /tmp/openclaw-relay.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable openclaw-relay
sudo systemctl start openclaw-relay

# Open firewall
sudo iptables -I INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 8443 -j ACCEPT
sudo netfilter-persistent save 2>/dev/null || true

echo "=== Setup complete ==="
echo "Check status: sudo systemctl status openclaw-relay"
