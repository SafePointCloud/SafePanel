# SafePanel Architecture

## Overview

SafePanel is a server panel focused on network traffic analysis and log analysis. It provides real-time monitoring, analysis, and protection capabilities through a modular and extensible architecture.

## Core Components

### 1. Analyzer Module (`internal/analyzer/`)

The analyzer module is responsible for collecting and analyzing different types of data.

#### Network Analysis (`analyzer/network/`)
- **IP Analysis**: Tracks IP connections, bandwidth usage, and connection patterns
- **DNS Analysis**: Monitors DNS queries and responses
- **Application Layer Analysis**: Analyzes specific protocols (MySQL, PostgreSQL, Redis)

#### Log Analysis (`analyzer/log/`)
- **Web Server Logs**: Processes Nginx, Apache logs
- **SSH Logs**: Analyzes SSH access patterns and authentication attempts

### 2. Blocker Module (`internal/blocker/`)

Handles the blocking of malicious or suspicious traffic.

- **IP Blocking**: Manages IP blacklists and implements blocking through firewall rules
- **DNS Blocking**: Blocks access to malicious domains
- **Integration**: Interfaces with system firewalls (iptables, nftables)

### 3. Alert Module (`internal/alert/`)

Manages the alert system and notification delivery.

- **Rule Engine**: Processes alert rules and triggers
- **Notification System**: Handles different notification channels (email, webhook)
- **Alert History**: Maintains a record of past alerts

### 4. Storage Module (`internal/storage/`)

Handles data persistence and retrieval.

- **Memory Storage**: Fast, in-memory storage for real-time data
- **Database Storage**: Persistent storage for historical data and configurations
- **Interface**: Common interface for different storage backends

## Data Flow

1. **Data Collection**
   - Network packets are captured by network analyzers
   - Log files are monitored by log analyzers
   - System metrics are collected periodically

2. **Analysis Pipeline**
   ```
   Raw Data → Analyzer → Stats/Events → Rule Engine → Actions/Alerts
   ```

3. **Action Flow**
   ```
   Trigger → Alert Generation → Notification → Action (e.g., Blocking)
   ```

## Configuration

Configuration is managed through YAML files in the `configs/` directory:

- `config.yaml`: Main configuration file
- `rules.yaml`: Alert rules configuration
- `blocklist.yaml`: Default blocking rules

## Extension Points

The system is designed to be extensible through several interfaces:

1. **Custom Analyzers**
   - Implement the Analyzer interface
   - Register with the analyzer registry

2. **Storage Backends**
   - Implement the Storage interface
   - Configure in storage settings

3. **Alert Channels**
   - Implement the Notifier interface
   - Add to notification configuration

## Security Considerations

1. **Access Control**
   - Role-based access control for web interface
   - API authentication and authorization
   - Audit logging for all actions

2. **Data Protection**
   - Encryption for sensitive configuration
   - Secure storage of credentials
   - Data retention policies

3. **System Security**
   - Minimal privilege principle
   - Secure defaults
   - Regular security updates

## Deployment

### Requirements
- Go 1.20 or higher
- Linux kernel 3.10+ (for network capture)

### Directory Structure
