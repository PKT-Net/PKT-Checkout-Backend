# PKT Checkout - Backend

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://Copyfree.org)
![Main branch build Status](https://github.com/PKT-Net/PKT-Checkout-Backend/actions/workflows/go.yml/badge.svg?branch=main)

This open-source project develops a generic backend and frontend for a payment processor that interfaces with pktwallet and can be self-hosted to accept payments settled in PKT. At present, the project is considered to be in alpha-release status and many features are currently still active WIP (work in progress).

## Working features

* Creating invoices to accept payments settled in absolute PKT amounts
* Discovery of (possibly several) transactions made towards an invoice
* Allow passing IPN-callback URL on invoice creation call

## Pending features

* Allow specifying notional values with integrated price oracle
* Automated sending of payments towards cold wallets
* Additional interfaces for third-party integrations (i.e. credit card vendors)
* Better error-handling, unexpected behaviour recovery, refactoring for consistency

## Configuration

```
# API
api-http-address: 127.0.0.1       # Set to 0.0.0.0 for access from different machines
api-http-port: 5000               # Any port can be configured
api-invoice-timeout: 15           # Minutes to wait before expiring an invoice without payment
api-cors-origin: https://test.com # URL for frontend to add necessary CORS headers

# MySQL
mysql-address: 127.0.0.1          # MySQL Server
mysql-port: 3306
mysql-database: pktcheckout       # Replace with your own credentials
mysql-user: pktcheckout
mysql-pass: pktcheckout

# Wallet
wallet-rpc-address: localhost     # Server hosting pktwallet instance
wallet-rpc-port: 8332             # Default port is 64763
wallet-rpc-user: x                # Specify with --rpcuser and --rpcpass
wallet-rpc-pass: x
wallet-addresses: 50              # Amount of addresses to generate to recycle
wallet-confirmations: 10          # Amount of blockchain confirmations to wait before trusting transactions

# Callback
callback-attempts: 5              # Amount of attempts to re-try a failed callback
callback-backoff: 10              # Minutes to back-off after failed attempt (attempts * backoff)
```

## Database scheme

```
CREATE TABLE `accounts` (
  `id` int(10) UNSIGNED NOT NULL,
  `merchant` varchar(32) NOT NULL,
  `apiKey` varchar(36) NOT NULL,
  `viewKey` varchar(36) NOT NULL,
  `secretKey` varchar(36) NOT NULL,
  `coldWallet` varchar(43) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `callbacks` (
  `id` varchar(36) NOT NULL,
  `invoiceId` varchar(36) NOT NULL,
  `requestTime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `nextReqTime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `reqErrors` int(11) NOT NULL,
  `status` varchar(16) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `invoices` (
  `id` varchar(36) NOT NULL,
  `clientId` varchar(36) NOT NULL,
  `accountId` int(11) NOT NULL,
  `paymentAmount` double NOT NULL,
  `paymentAddress` varchar(43) NOT NULL,
  `paymentDescription` varchar(64) NOT NULL,
  `callbackUrl` varchar(64) DEFAULT NULL,
  `creationTime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `expirationTime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `status` varchar(16) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `walletAddresses` (
  `address` varchar(43) NOT NULL,
  `lastUsed` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `inUse` tinyint(1) NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `walletTransactions` (
  `id` varchar(64) NOT NULL,
  `invoiceId` varchar(36) NOT NULL,
  `walletAddress` varchar(43) NOT NULL,
  `paymentAmount` double UNSIGNED NOT NULL,
  `confirmationTime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `discoveryTime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

ALTER TABLE `accounts`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `apiKey` (`apiKey`),
  ADD UNIQUE KEY `viewKey` (`viewKey`),
  ADD UNIQUE KEY `secretKey` (`secretKey`),
  ADD UNIQUE KEY `coldWallet` (`coldWallet`);

ALTER TABLE `callbacks`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `invoiceId` (`invoiceId`);

ALTER TABLE `invoices`
  ADD PRIMARY KEY (`id`),
  ADD KEY `id` (`id`,`paymentAddress`);

ALTER TABLE `walletAddresses`
  ADD PRIMARY KEY (`address`);

ALTER TABLE `walletTransactions`
  ADD PRIMARY KEY (`id`),
  ADD KEY `id` (`id`,`invoiceId`);

ALTER TABLE `accounts`
  MODIFY `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=3;
COMMIT;
```

## Installation (Debian/Ubuntu)

#### Clone the repository

```
git clone https://github.com/pkt-net/pkt-checkout-backend.git /opt/pkt-checkout-backend
```

#### Install dependencies

```
apt update
apt install mariadb-server mariadb-client nginx
wget https://go.dev/dl/go1.22.4.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.22.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

#### Compile from source

```
cd /opt/pkt-checkout-backend/cmd
go build main.go
```

#### Install blockchain dependencies

```
git clone https://github.com/pkt-cash/pktd.git /opt/pktd
cd /opt/pktd
./do
```

#### Create a new wallet or restore existing one

```
/opt/pktd/bin/pktwallet --create
```

#### /etc/systemd/system/pktwallet.service

```
[Unit]
Description=PKT Wallet
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=always
RestartSec=5
WorkingDirectory=/opt/pktd
ExecStart=/opt/pktd/bin/pktwallet --rpclisten 127.0.0.1:8332 --rpcuser x --rpcuser x

[Install]
WantedBy=multi-user.target
```

#### /etc/systemd/system/pkt-checkout-backend.service

```
[Unit]
Description=PKT Checkout Backend
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=always
RestartSec=5
WorkingDirectory=/opt/pkt-checkout-backend
ExecStart=/opt/pkt-checkout-backend/cmd/main

[Install]
WantedBy=multi-user.target
```

#### /etc/nginx/sites-enabled/pkt-checkout-backend

```
server {
    ...
    
        location / {
                proxy_set_header X-Real-IP $remote_addr;
                proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
                proxy_pass http://127.0.0.1:5000;
        }
    
    ...
}
```

#### Restore database scheme

```
$ mysql
MariaDB [(none)]> CREATE USER 'pktcheckout'@'127.0.0.1' IDENTIFIED BY 'pktcheckout';
MariaDB [(none)]> CREATE DATABASE pktcheckout;
MariaDB [(none)]> GRANT ALL PRIVILEGES ON pktcheckout.* TO 'pktcheckout'@'127.0.0.1';
MariaDB [(none)]> USE pktcheckout;
MariaDB [pktcheckout]> # Copy & paste database scheme from above
MariaDB [pktcheckout]> exit;
```

#### Starting all services

```
systemctl enable pktwallet
systemctl enable pkt-checkout-backend
systemctl start pktwallet
systemctl start pkt-checkout-backend
systemctl restart nginx
```

#### Example usage

```
# paymentAmount is denominated in µPKT - lowest precision is 1 µPKT
# Signature is the sha256 hmac of the request body with secretKey
curl -X POST http://127.0.0.1:5000/v1/invoices -H 'X-API-KEY: 679aa2f2-2072-4867-9216-2719139103c6' -H 'X-SIGNATURE: 5a5f9de2647fbaaca78df7ad453a31ba1a513dee154dab891c7acad8fc5073f0' -d '{"clientId":"invoice-1337","paymentAmount":1000,"paymentDescription":"3 months of VPN service","callbackUrl":"https://myawesomeservice.com/pkt-ipn"}'
```
```
{"id":"7a1ac6c2-98fd-4055-a4e1-4a2d0bd17421","clientId":"invoice-1337","accountId":2,"paymentAmount":1000,"paymentAddress":"pkt1q4h38kq2rzcz92h7hwexjkztv72dv9w32l72azm","paymentDescription":"3 months of VPN service","callbackUrl":"https://myawesomeservice.com/pkt-ipn","creationTime":"2024-06-15T22:40:04.193226591Z","expirationTime":"2024-06-15T22:55:04.193226641Z","status":"created"}
```