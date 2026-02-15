#!/bin/bash
set -e

INSTALL_DIR="/usr/local/bin"
LIB_DIR="/lib"
CONFIG_DIR="/etc/nss-daemon"
SYSTEMD_DIR="/etc/systemd/system"
NSSWITCH_FILE="/etc/nsswitch.conf"

echo "=== NSS Daemon Uninstallation Script ==="
echo

# Check if running as root
if [[ "${EUID}" -ne 0 ]]; then
    echo "Error: Please run as root or with sudo"
    exit 1
fi

# Stop and disable systemd service if installed
if [[ -f "${SYSTEMD_DIR}/nss-daemon.service" ]]; then
    echo "Stopping nss-daemon service..."
    systemctl stop nss-daemon 2>/dev/null || true
    systemctl disable nss-daemon 2>/dev/null || true
    rm -f "${SYSTEMD_DIR}/nss-daemon.service"
    systemctl daemon-reload
    echo "Removed: ${SYSTEMD_DIR}/nss-daemon.service"
fi

# Remove binaries
echo "Removing binaries..."
if [[ -f "${INSTALL_DIR}/nss-daemon" ]]; then
    rm -f "${INSTALL_DIR}/nss-daemon"
    echo "Removed: ${INSTALL_DIR}/nss-daemon"
fi
if [[ -f "${INSTALL_DIR}/nss-status" ]]; then
    rm -f "${INSTALL_DIR}/nss-status"
    echo "Removed: ${INSTALL_DIR}/nss-status"
fi
if [[ -f "${INSTALL_DIR}/nss-query" ]]; then
    rm -f "${INSTALL_DIR}/nss-query"
    echo "Removed: ${INSTALL_DIR}/nss-query"
fi
if [[ -f "${INSTALL_DIR}/nss-key" ]]; then
    rm -f "${INSTALL_DIR}/nss-key"
    echo "Removed: ${INSTALL_DIR}/nss-key"
fi
if [[ -f "${INSTALL_DIR}/nss-config-validate" ]]; then
    rm -f "${INSTALL_DIR}/nss-config-validate"
    echo "Removed: ${INSTALL_DIR}/nss-config-validate"
fi

if [[ -f "${INSTALL_DIR}/nss-ping" ]]; then
    rm -f "${INSTALL_DIR}/nss-ping"
    echo "Removed: ${INSTALL_DIR}/nss-ping"
fi

if [[ -f "${INSTALL_DIR}/nss-dns" ]]; then
    rm -f "${INSTALL_DIR}/nss-dns"
    echo "Removed: ${INSTALL_DIR}/nss-dns"
fi

# Remove NSS module
if [[ -f "${LIB_DIR}/libnss_daemon.so.2" ]]; then
    echo "Removing NSS module..."
    rm -f "${LIB_DIR}/libnss_daemon.so.2"
    rm -f "${LIB_DIR}/libnss_daemon.so"
    ldconfig
    echo "Removed: ${LIB_DIR}/libnss_daemon.so.2"
fi

# Remove configuration directory (prompt user)
if [[ -d "${CONFIG_DIR}" ]]; then
    read -p "Remove configuration directory ${CONFIG_DIR}? [y/N] " -n 1 -r
    echo

    if [[ ${REPLY} =~ ^[Yy]$ ]]; then
        rm -rf "${CONFIG_DIR}"
        echo "Removed: ${CONFIG_DIR}"
    else
        echo "Keeping: ${CONFIG_DIR}"
    fi
fi

# Remove state directory (prompt user)
if [[ -d "/var/lib/nss-daemon" ]]; then
    read -p "Remove state directory /var/lib/nss-daemon? [y/N] " -n 1 -r
    echo
    if [[ "${REPLY}" =~ ^[Yy]$ ]]; then
        rm -rf "/var/lib/nss-daemon"
        echo "Removed: /var/lib/nss-daemon"
    else
        echo "Keeping: /var/lib/nss-daemon"
    fi
fi

# Remove log file (prompt user)
if [[ -f "/var/log/nss-daemon.log" ]]; then
    read -p "Remove log file /var/log/nss-daemon.log? [y/N] " -n 1 -r
    echo
    if [[ ${REPLY} =~ ^[Yy]$ ]]; then
        rm -f "/var/log/nss-daemon.log"
        echo "Removed: /var/log/nss-daemon.log"
    else
        echo "Keeping: /var/log/nss-daemon.log"
    fi
fi

# Remove logrotate config
if [[ -f "/etc/logrotate.d/nss-daemon" ]]; then
    rm -f "/etc/logrotate.d/nss-daemon"
    echo "Removed: /etc/logrotate.d/nss-daemon"
fi

# Remove nss-daemon user (prompt user)
if id -u nss-daemon &>/dev/null; then
    read -p "Remove nss-daemon user and group? [y/N] " -n 1 -r
    echo
    if [[ "${REPLY}" =~ ^[Yy]$ ]]; then
        userdel nss-daemon
        groupdel nss-daemon 2>/dev/null || true
        echo "Removed: nss-daemon user and group"
    else
        echo "Keeping: nss-daemon user and group"
    fi
fi

# Remove from nsswitch.conf
echo
echo "Updating nsswitch.conf..."
if [[ -f "${NSSWITCH_FILE}" ]]; then
    if grep -q "daemon" "${NSSWITCH_FILE}"; then
        cp "${NSSWITCH_FILE}" "${NSSWITCH_FILE}.backup.$(date +'%Y%m%d%H%M%S' || true)"

        # Remove 'daemon' from hosts line
        sed -i.bak 's/ daemon//' "${NSSWITCH_FILE}"

        # Restore from backup if sed created it
        if [[ -f "${NSSWITCH_FILE}.bak" ]]; then
            rm "${NSSWITCH_FILE}.bak"
        fi

        echo "Updated: ${NSSWITCH_FILE}"
        echo "Backup created: ${NSSWITCH_FILE}.backup.*"
    else
        echo "NSSwitch already configured (no 'daemon' entry)"
    fi
else
    echo "Warning: ${NSSWITCH_FILE} not found"
fi

echo
echo "=== Uninstallation Complete ==="
echo
echo "Note: Some files may have been kept based on your choices"
echo "Backups of modified files have been created with timestamp suffixes"
