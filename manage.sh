#!/bin/bash

# 定义颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color
YELLOW='\033[1;33m'

# define variables
INSTALL_DIR="/usr/local/safepanel"
CONFIG_DIR="/etc/safepanel"
LOG_DIR="/var/log/safepanel"
BINARY_NAME="safepaneld"
SP_STATS_NAME="sp-stats"
SP_BLOCKER_NAME="sp-blocker"
BINARY_PATH="/usr/local/bin/${BINARY_NAME}"
SP_STATS_PATH="/usr/local/bin/${SP_STATS_NAME}"
SP_BLOCKER_PATH="/usr/local/bin/${SP_BLOCKER_NAME}"
SERVICE_NAME="safepaneld"
GITHUB_REPO="safepointcloud/safepanel"

# ensure running with root privileges
check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo -e "${RED}Please run this script with root privileges${NC}"
        exit 1
    fi
}

# create necessary directories
create_directories() {
    echo -e "${GREEN}Creating necessary directories...${NC}"
    mkdir -p "${INSTALL_DIR}"
    mkdir -p "${CONFIG_DIR}"
    mkdir -p "${LOG_DIR}"
}

# detect system architecture
detect_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "x86_64"
            ;;
        aarch64)
            echo "arm64"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $arch${NC}"
            exit 1
            ;;
    esac
}

# download latest version
download_latest() {
    echo -e "${GREEN}Downloading latest version...${NC}"
    # get latest version
    latest_version=$(curl -s https://api.github.com/repos/${GITHUB_REPO}/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$latest_version" ]; then
        echo -e "${RED}Failed to get version information${NC}"
        exit 1
    fi

    # Detect architecture
    ARCH=$(detect_arch)
    echo -e "${GREEN}Detected architecture: $ARCH${NC}"

    # Create temporary directory for extraction
    TMP_DIR=$(mktemp -d)

    # Download and extract archive
    echo -e "${GREEN}Downloading SafePanel_Linux_${ARCH}.tar.gz...${NC}"
    if ! curl -L "https://github.com/${GITHUB_REPO}/releases/download/${latest_version}/SafePanel_Linux_${ARCH}.tar.gz" -o "${TMP_DIR}/safepanel.tar.gz"; then
        echo -e "${RED}Download failed${NC}"
        rm -rf "${TMP_DIR}"
        exit 1
    fi

    # Extract files
    if ! tar -xzf "${TMP_DIR}/safepanel.tar.gz" -C "${TMP_DIR}"; then
        echo -e "${RED}Extraction failed${NC}"
        rm -rf "${TMP_DIR}"
        exit 1
    fi

    # Move binary and database files to their destinations
    mv "${TMP_DIR}/safepaneld" "${BINARY_PATH}"
    mv "${TMP_DIR}/sp-stats" "${SP_STATS_PATH}"
    mv "${TMP_DIR}/sp-blocker" "${SP_BLOCKER_PATH}"
    chmod +x "${BINARY_PATH}"
    chmod +x "${SP_STATS_PATH}"
    chmod +x "${SP_BLOCKER_PATH}"
    mv "${TMP_DIR}/ip-threat.db" "${INSTALL_DIR}/ip-threat.db"
    mv "${TMP_DIR}/GeoLite2-Country.mmdb" "${INSTALL_DIR}/GeoLite2-Country.mmdb"

    # Cleanup
    rm -rf "${TMP_DIR}"
}

# generate config file
generate_config() {
    echo -e "${GREEN}Generating config file...${NC}"
    # Get default interface
    DEFAULT_INTERFACE=$(ip route | grep default | awk '{print $5}' | head -n1)
    if [ -z "$DEFAULT_INTERFACE" ]; then
        echo -e "${YELLOW}Warning: Could not detect default interface, falling back to eth0${NC}"
        DEFAULT_INTERFACE="eth0"
    fi
    cat > "${CONFIG_DIR}/config.yaml" << EOF
analyzer:
  network:
    ip:
      interface: "${DEFAULT_INTERFACE}"

checker:
  ipdb_path: "${INSTALL_DIR}/ip-threat.db"
  mmdb_path: "${INSTALL_DIR}/GeoLite2-Country.mmdb"

EOF
}

# 生成 systemd 服务文件
generate_service() {
    echo -e "${GREEN}Generating systemd service file...${NC}"
    cat > "/etc/systemd/system/${SERVICE_NAME}.service" << EOF
[Unit]
Description=SafePanel Daemon
After=network.target

[Service]
Type=simple
ExecStart=${BINARY_PATH}
Restart=always
RestartSec=10
User=root
Group=root
Environment=CONFIG_PATH=${CONFIG_DIR}/config.yaml
ProtectSystem=full
ProtectHome=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
}

# start service
start_service() {
    echo -e "${GREEN}Starting service...${NC}"
    systemctl start ${SERVICE_NAME}
    systemctl enable ${SERVICE_NAME}
    echo -e "${GREEN}Service started and set to start on boot${NC}"
}

# stop service
stop_service() {
    echo -e "${YELLOW}Stopping service...${NC}"
    systemctl stop ${SERVICE_NAME}
    systemctl disable ${SERVICE_NAME}
    echo -e "${YELLOW}Service stopped and disabled from starting on boot${NC}"
}

# check service status
status_service() {
    systemctl status ${SERVICE_NAME}
}

# view logs
view_logs() {
    journalctl -u ${SERVICE_NAME} -f
}

# uninstall
uninstall() {
    echo -e "${YELLOW}Starting uninstall...${NC}"
    stop_service
    rm -f "${BINARY_PATH}"
    rm -f "${SP_STATS_PATH}"
    rm -f "${SP_BLOCKER_PATH}"
    rm -rf "${INSTALL_DIR}"
    rm -rf "${CONFIG_DIR}"
    rm -rf "${LOG_DIR}"
    rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
    systemctl daemon-reload
    echo -e "${GREEN}Uninstall completed${NC}"
}

# show help
show_help() {
    echo "SafePanel management script"
    echo "Usage: $0 [command]"
    echo
    echo "Available commands:"
    echo "  install     Install or update SafePanel"
    echo "  start       Start the service"
    echo "  stop        Stop the service"
    echo "  status      Check service status"
    echo "  logs        View service logs"
    echo "  uninstall   Uninstall SafePanel"
    echo "  help        Display this help information"
}

# 主函数
main() {
    check_root

    case "$1" in
        "install")
            create_directories
            download_latest
            generate_config
            generate_service
            start_service
            ;;
        "start")
            start_service
            ;;
        "stop")
            stop_service
            ;;
        "status")
            status_service
            ;;
        "logs")
            view_logs
            ;;
        "uninstall")
            uninstall
            ;;
        "help"|"")
            show_help
            ;;
        *)
            echo -e "${RED}未知命令: $1${NC}"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
