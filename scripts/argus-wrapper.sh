#!/bin/bash
# =============================================================================
# Argus Secure Wrapper Script
# =============================================================================
#
# SECURITY MODEL:
# ===============
# This script is the ONLY entry point for running Argus with elevated privileges.
# It is owned by root:root with mode 755 (not writable by anyone but root).
#
# The sudoers rule allows members of the 'argus-users' group to run ONLY this
# script without a password. This prevents:
#   1. Users modifying the script to run arbitrary commands as root
#   2. Privilege escalation through argument injection
#   3. Direct access to the Python/Go binary with arbitrary flags
#
# INSTALLATION:
# =============
# 1. Copy this script to /usr/local/bin/argus
#    sudo cp scripts/argus-wrapper.sh /usr/local/bin/argus
#
# 2. Set ownership and permissions
#    sudo chown root:root /usr/local/bin/argus
#    sudo chmod 755 /usr/local/bin/argus
#
# 3. Create the argus-users group
#    sudo groupadd argus-users
#    sudo usermod -aG argus-users $USER
#
# 4. Add sudoers rule (create /etc/sudoers.d/argus)
#    %argus-users ALL=(ALL) NOPASSWD: /usr/local/bin/argus
#
# 5. Log out and back in for group membership to take effect
#
# USAGE:
# ======
# Users run: sudo argus [options]
# No password will be prompted if they're in the argus-users group.
# =============================================================================

set -euo pipefail

# Hardcoded path to the actual binary
# This CANNOT be overridden by users
ARGUS_BIN="/usr/local/bin/argus-bin"

# Verify the binary exists and is owned by root
if [[ ! -f "$ARGUS_BIN" ]]; then
    echo "Error: Argus binary not found at $ARGUS_BIN" >&2
    echo "Please run 'make install' as root first." >&2
    exit 1
fi

# Verify ownership (security check)
OWNER=$(stat -c '%U' "$ARGUS_BIN")
if [[ "$OWNER" != "root" ]]; then
    echo "Security Error: $ARGUS_BIN is not owned by root!" >&2
    echo "This is a security violation. Please reinstall." >&2
    exit 1
fi

# Execute the binary with passed arguments
# We use exec to replace this shell process (no lingering shell)
exec "$ARGUS_BIN" "$@"
