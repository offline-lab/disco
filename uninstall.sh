#!/bin/bash
set -e

INSTALL_DIR="/usr/local/bin"
LIB_DIR="/lib"
CONFIG_DIR="/etc/disco"
SYSTEMD_DIR="/etc/systemd/system"
NSSWITCH_FILE="/etc/nsswitch.conf"

echo "=== Disco Daemon Uninstallation Script ==="
echo

if [[ "${EUID}" -ne 0 ]]; then
    echo "Error: Please run as root or with sudo"
    exit 1
fi

if [[ -f "${SYSTEMD_DIR}/disco.service" ]]; then
    echo "Stopping disco service..."
    systemctl stop disco 2>/dev/null || true
    systemctl disable disco 2>/dev/null || true
    rm -f "${SYSTEMD_DIR}/disco.service"
    systemctl daemon-reload
    echo "Removed: ${SYSTEMD_DIR}/disco.service"
fi

echo "Removing binaries..."
for bin in disco disco-daemon disco-gps-broadcaster; do
    if [[ -f "${INSTALL_DIR}/${bin}" ]]; then
        rm -f "${INSTALL_DIR}/${bin}"
        echo "Removed: ${INSTALL_DIR}/${bin}"
    fi
done

for old_bin in disco-status disco-query disco-key disco-ping disco-dns disco-config-validate disco-time disco-timeset disco-announce; do
    if [[ -f "${INSTALL_DIR}/${old_bin}" ]]; then
        rm -f "${INSTALL_DIR}/${old_bin}"
        echo "Removed (old): ${INSTALL_DIR}/${old_bin}"
    fi
done

if [[ -f "${LIB_DIR}/libnss_disco.so.2" ]]; then
    echo "Removing NSS module..."
    rm -f "${LIB_DIR}/libnss_disco.so.2"
    rm -f "${LIB_DIR}/libnss_disco.so"
    ldconfig
    echo "Removed: ${LIB_DIR}/libnss_disco.so.2"
fi

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

if [[ -d "/var/lib/disco" ]]; then
    read -p "Remove state directory /var/lib/disco? [y/N] " -n 1 -r
    echo
    if [[ "${REPLY}" =~ ^[Yy]$ ]]; then
        rm -rf "/var/lib/disco"
        echo "Removed: /var/lib/disco"
    else
        echo "Keeping: /var/lib/disco"
    fi
fi

if [[ -f "/var/log/disco.log" ]]; then
    read -p "Remove log file /var/log/disco.log? [y/N] " -n 1 -r
    echo
    if [[ ${REPLY} =~ ^[Yy]$ ]]; then
        rm -f "/var/log/disco.log"
        echo "Removed: /var/log/disco.log"
    else
        echo "Keeping: /var/log/disco.log"
    fi
fi

if [[ -f "/etc/logrotate.d/disco" ]]; then
    rm -f "/etc/logrotate.d/disco"
    echo "Removed: /etc/logrotate.d/disco"
fi

if id -u disco &>/dev/null; then
    read -p "Remove disco user and group? [y/N] " -n 1 -r
    echo
    if [[ "${REPLY}" =~ ^[Yy]$ ]]; then
        userdel disco
        groupdel disco 2>/dev/null || true
        echo "Removed: disco user and group"
    else
        echo "Keeping: disco user and group"
    fi
fi

echo
echo "Updating nsswitch.conf..."
if [[ -f "${NSSWITCH_FILE}" ]]; then
    if grep -q "disco" "${NSSWITCH_FILE}"; then
        cp "${NSSWITCH_FILE}" "${NSSWITCH_FILE}.backup.$(date +'%Y%m%d%H%M%S' || true)"

        sed -i.bak 's/ disco//' "${NSSWITCH_FILE}"

        if [[ -f "${NSSWITCH_FILE}.bak" ]]; then
            rm "${NSSWITCH_FILE}.bak"
        fi

        echo "Updated: ${NSSWITCH_FILE}"
        echo "Backup created: ${NSSWITCH_FILE}.backup.*"
    else
        echo "NSSwitch already configured (no 'disco' entry)"
    fi
else
    echo "Warning: ${NSSWITCH_FILE} not found"
fi

echo
echo "=== Uninstallation Complete ==="
echo
echo "Note: Some files may have been kept based on your choices"
echo "Backups of modified files have been created with timestamp suffixes"
