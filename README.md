# SafePanel

SafePanel is a server panel focused on network traffic analysis and log analysis.

## Features

- Network traffic analysis and blocking
    - Analysis
        - Connection tracking
        - DNS analysis
        - Application layer analysis(MySQL, PostgreSQL, Redis, etc.)
    - Blocking
        - IP blocking
        - DNS blocking
- Log analysis
    - Web server log analysis(Nginx, Apache, etc.)
    - SSH log analysis
- Alerting
    - Alerting rules
    - Alerting notifications

## Installation

### Install SafePanel Daemon

You can install SafePanel Daemon using our installation script:

```bash
curl -O https://raw.githubusercontent.com/SafePointCloud/SafePanel/main/manage.sh
chmod +x manage.sh
sudo ./manage.sh install
```

To manage SafePanel, you can use the following commands:

```bash
sudo ./manage.sh start     # Start the service
sudo ./manage.sh stop      # Stop the service
sudo ./manage.sh status    # Check service status
sudo ./manage.sh logs      # View service logs
sudo ./manage.sh uninstall # Uninstall SafePanel
```

### Using TUI

#### sp-blocker
sp-blocker is an interactive TUI tool for managing network blacklists and blocking rules:

```bash
sudo sp-blocker
```
![image](https://github.com/user-attachments/assets/c2fff01e-3aa6-44c7-87be-bbe45c7db76c)

#### sp-stats
sp-stats is an interactive TUI tool for viewing network traffic statistics:

```bash
sudo sp-stats
```
![image](https://github.com/user-attachments/assets/afbf33a2-9f89-4eb4-9ab3-03de81740457)


## License

SafePanel is licensed under the GNU General Public License v3.0. See the [LICENSE](https://github.com/SafePointCloud/SafePanel/blob/main/LICENSE) file for more details.
