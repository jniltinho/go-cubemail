# go-cubemail — Installation Guide (Ubuntu)

## Prerequisites

- Ubuntu 22.04 or 24.04
- MariaDB 10.11+

---

## 1. System Dependencies

```bash
sudo apt update && sudo apt install -y curl wget mariadb-server mariadb-client
```

---

## 2. Configure MariaDB

### 2.1 Start and Enable the Service

```bash
sudo systemctl enable --now mariadb
sudo mariadb-secure-installation
```

### 2.2 Create Database and User

```bash
sudo mariadb -u root -p
```

```sql
CREATE DATABASE cubemail CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'cubemail'@'localhost' IDENTIFIED BY 'strong_password_here';
GRANT ALL PRIVILEGES ON cubemail.* TO 'cubemail'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

---

## 3. Install go-cubemail from Release

### 3.1 Download the Binary

Access the releases page: https://github.com/jniltinho/go-cubemail/releases

```bash
VERSION=0.0.1
cd /tmp
wget https://github.com/jniltinho/go-cubemail/releases/download/v${VERSION}/go-cubemail_${VERSION}_linux_amd64.tar.gz
```

### 3.2 Extract and Install in `/opt/go-cubemail`

```bash
sudo mkdir -p /opt/go-cubemail
sudo tar -xzf /tmp/go-cubemail_${VERSION}_linux_amd64.tar.gz -C /opt/go-cubemail
sudo chmod +x /opt/go-cubemail/go-cubemail
```

### 3.3 Verify Installation

```bash
/opt/go-cubemail/go-cubemail version
```

---

## 4. Configuration

### 4.1 Create the Configuration File in `/opt/go-cubemail`

```bash
sudo tee /opt/go-cubemail/config.toml > /dev/null << 'EOF'
[server]
host       = "0.0.0.0"
port       = 8080
debug      = false
secret_key = "change-to-a-32-character-key!!"
base_url   = "https://webmail.yourdomain.com.br"

[imap]
host            = "mail.yourdomain.com.br"
port            = 993
tls             = true
timeout_sec     = 30
show_host_input = false

[smtp]
host        = "mail.yourdomain.com.br"
port        = 587
starttls    = true
timeout_sec = 30

[database]
driver = "mariadb"
dsn    = "cubemail:strong_password_here@tcp(localhost:3306)/cubemail?charset=utf8mb4&parseTime=True&loc=Local"

[session]
name      = "gorc_session"
max_age   = 86400
secure    = true
http_only = true

[ui]
rows_per_page   = 50
timezone        = "America/Sao_Paulo"
date_format     = "02/01/2006"
datetime_format = "02/01/2006 15:04"
compose_html    = true

[upload]
max_size_mb = 25
temp_dir    = "/tmp/go-cubemail-uploads"
EOF
```

### 4.2 Adjust the Configuration File

```bash
sudo nano /opt/go-cubemail/config.toml
```

Required substitutions:
- `secret_key` — random key with 32+ characters (use `openssl rand -hex 16`)
- `imap.host` / `smtp.host` — email server
- `database.dsn` — database password created in step 2.2

---

## 5. Run Migrations

Automatically creates tables in MariaDB:

```bash
/opt/go-cubemail/go-cubemail migrate
```

Tables created: `sessions`, `users`, `user_settings`, `identities`, `contacts`, `contact_groups`, `drafts`

---

## 6. Test Manually

```bash
cd /opt/go-cubemail
./go-cubemail serve
```

Access: `http://localhost:8080`

---

## 7. Systemd Service

### 7.1 Copy the Service File

```bash
sudo cp /opt/go-cubemail/DOCUMENTS/setup/cubemail.service /etc/systemd/system/go-cubemail.service
```

Or create manually:

```bash
sudo tee /etc/systemd/system/go-cubemail.service > /dev/null << 'EOF'
[Unit]
Description=go-cubemail Webmail
Documentation=https://github.com/jniltinho/go-cubemail
After=network.target mariadb.service
Wants=mariadb.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/go-cubemail
ExecStart=/opt/go-cubemail/go-cubemail serve
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=5
TimeoutStopSec=30
StandardOutput=journal
StandardError=journal
SyslogIdentifier=go-cubemail

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/tmp/go-cubemail-uploads

[Install]
WantedBy=multi-user.target
EOF
```

### 7.2 Enable and Start

```bash
sudo chown -R www-data:www-data /opt/go-cubemail
sudo mkdir -p /tmp/go-cubemail-uploads
sudo chown www-data:www-data /tmp/go-cubemail-uploads

sudo systemctl daemon-reload
sudo systemctl enable --now go-cubemail
sudo systemctl status go-cubemail
```

---

## 8. Update

```bash
VERSION=0.0.2   # new version
cd /tmp
wget https://github.com/jniltinho/go-cubemail/releases/download/v${VERSION}/go-cubemail_${VERSION}_linux_amd64.tar.gz

sudo systemctl stop go-cubemail
sudo tar -xzf go-cubemail_${VERSION}_linux_amd64.tar.gz -C /opt/go-cubemail
sudo chmod +x /opt/go-cubemail/go-cubemail
sudo chown www-data:www-data /opt/go-cubemail/go-cubemail

/opt/go-cubemail/go-cubemail migrate
sudo systemctl start go-cubemail
sudo systemctl status go-cubemail
```

---

## 9. Environment Variables (Alternative to config.toml)

All `config.toml` options can be set via environment variables with `GORC_` prefix:

```bash
export GORC_DATABASE_DRIVER=mariadb
export GORC_DATABASE_DSN="cubemail:password@tcp(localhost:3306)/cubemail?charset=utf8mb4&parseTime=True&loc=Local"
export GORC_SERVER_SECRET_KEY="32-character-key-here!!!!"
export GORC_IMAP_HOST=mail.yourdomain.com.br
export GORC_SMTP_HOST=mail.yourdomain.com.br
```

---

## Production Directory Structure

```
/opt/go-cubemail/
├── go-cubemail        ← binary
└── config.toml        ← configuration
```

---

## Troubleshooting

| Problem | Solution |
|---|---|
| `dial tcp: connection refused` | Verify IMAP/SMTP host/port in config.toml |
| `Error 1049: Unknown database` | Create database according to step 2.2 |
| `permission denied` | `sudo chown -R www-data /opt/go-cubemail` |
| Blank screen / no style | Release version includes compiled CSS — check tar.gz integrity |
| Service logs | `sudo journalctl -u go-cubemail -f` |
| Test MariaDB connection | `mariadb -u cubemail -p cubemail -e "SHOW TABLES;"` |
