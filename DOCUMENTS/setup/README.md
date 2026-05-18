# go-cubemail — Guia de Instalação (Ubuntu)

## Pré-requisitos

- Ubuntu 22.04 ou 24.04
- MariaDB 10.11+

---

## 1. Dependências do sistema

```bash
sudo apt update && sudo apt install -y \
    curl wget mariadb-server mariadb-client
```

---

## 2. Configurar o MariaDB

### 2.1 Iniciar e habilitar o serviço

```bash
sudo systemctl enable --now mariadb
sudo mariadb-secure-installation
```

### 2.2 Criar banco de dados e usuário

```bash
sudo mariadb -u root -p
```

```sql
CREATE DATABASE cubemail CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'cubemail'@'localhost' IDENTIFIED BY 'senha_forte_aqui';
GRANT ALL PRIVILEGES ON cubemail.* TO 'cubemail'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

---

## 3. Instalar o go-cubemail a partir do Release

### 3.1 Baixar o binário

Acesse a página de releases: https://github.com/jniltinho/go-cubemail/releases

```bash
VERSION=0.0.1
cd /tmp
wget https://github.com/jniltinho/go-cubemail/releases/download/v${VERSION}/go-cubemail_${VERSION}_linux_amd64.tar.gz
```

### 3.2 Extrair e instalar em `/opt/go-cubemail`

```bash
sudo mkdir -p /opt/go-cubemail
sudo tar -xzf /tmp/go-cubemail_${VERSION}_linux_amd64.tar.gz -C /opt/go-cubemail
sudo chmod +x /opt/go-cubemail/go-cubemail
```

### 3.3 Verificar instalação

```bash
/opt/go-cubemail/go-cubemail version
```

---

## 4. Configuração

### 4.1 Criar o arquivo de configuração em `/opt/go-cubemail`

```bash
sudo tee /opt/go-cubemail/config.toml > /dev/null << 'EOF'
[server]
host       = "0.0.0.0"
port       = 8080
debug      = false
secret_key = "troque-por-uma-chave-de-32-caracteres!!"
base_url   = "https://webmail.seudominio.com.br"

[imap]
host            = "mail.seudominio.com.br"
port            = 993
tls             = true
timeout_sec     = 30
show_host_input = false

[smtp]
host        = "mail.seudominio.com.br"
port        = 587
starttls    = true
timeout_sec = 30

[database]
driver = "mariadb"
dsn    = "cubemail:senha_forte_aqui@tcp(localhost:3306)/cubemail?charset=utf8mb4&parseTime=True&loc=Local"

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

### 4.2 Ajustar o arquivo de configuração

```bash
sudo nano /opt/go-cubemail/config.toml
```

Substitua obrigatoriamente:
- `secret_key` — chave aleatória de 32+ caracteres (use `openssl rand -hex 16`)
- `imap.host` / `smtp.host` — servidor de e-mail
- `database.dsn` — senha do banco criada no passo 2.2

---

## 5. Executar as migrations

Cria automaticamente as tabelas no MariaDB:

```bash
/opt/go-cubemail/go-cubemail migrate
```

Tabelas criadas: `sessions`, `users`, `user_settings`, `identities`, `contacts`, `contact_groups`, `drafts`

---

## 6. Testar manualmente

```bash
cd /opt/go-cubemail
./go-cubemail serve
```

Acesse: `http://localhost:8080`

---

## 7. Serviço systemd

### 7.1 Copiar o arquivo de serviço

```bash
sudo cp /opt/go-cubemail/DOCUMENTS/setup/cubemail.service /etc/systemd/system/go-cubemail.service
```

Ou criar manualmente:

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

# Segurança
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/tmp/go-cubemail-uploads

[Install]
WantedBy=multi-user.target
EOF
```

### 7.2 Habilitar e iniciar

```bash
sudo chown -R www-data:www-data /opt/go-cubemail
sudo mkdir -p /tmp/go-cubemail-uploads
sudo chown www-data:www-data /tmp/go-cubemail-uploads

sudo systemctl daemon-reload
sudo systemctl enable --now go-cubemail
sudo systemctl status go-cubemail
```

---

## 8. Atualização

```bash
VERSION=0.0.2   # nova versão
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

## 9. Variáveis de ambiente (alternativa ao config.toml)

Todas as opções do config.toml podem ser definidas via variáveis de ambiente com prefixo `GORC_`:

```bash
export GORC_DATABASE_DRIVER=mariadb
export GORC_DATABASE_DSN="cubemail:senha@tcp(localhost:3306)/cubemail?charset=utf8mb4&parseTime=True&loc=Local"
export GORC_SERVER_SECRET_KEY="chave-de-32-caracteres-aqui!!!!"
export GORC_IMAP_HOST=mail.seudominio.com.br
export GORC_SMTP_HOST=mail.seudominio.com.br
```

---

## Estrutura de arquivos em produção

```
/opt/go-cubemail/
├── go-cubemail        ← binário
└── config.toml        ← configuração
```

---

## Troubleshooting

| Problema | Solução |
|---|---|
| `dial tcp: connection refused` | Verificar host/porta do IMAP/SMTP no config.toml |
| `Error 1049: Unknown database` | Criar o banco conforme passo 2.2 |
| `permission denied` | `sudo chown -R www-data /opt/go-cubemail` |
| Tela em branco / sem estilo | Versão do release inclui CSS compilado — verificar integridade do tar.gz |
| Logs do serviço | `sudo journalctl -u go-cubemail -f` |
| Testar conexão MariaDB | `mariadb -u cubemail -p cubemail -e "SHOW TABLES;"` |
