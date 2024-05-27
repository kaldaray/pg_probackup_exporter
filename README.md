# PostgreSQL probackup exporter

Prometheus exporter for various metrics about pg_probackup, written in Go. Listen 9231 port/tcp.

## Metrics 

| Metric                            | Table    | Description                                                           |
| --------------------------------- |:--------:| ---------------------------------------------------------------------:|
| pg_probackup_integrity_check      | Gauge    | pg_probackup_integrity_check Integrity check status 1 - OK, 0 - FAIL  |
| pg_probackup_size_bytes           | Gauge    | Size of backup in bytes                                               |
| pg_probackup_status               | Gauge    | Status of current backup 1 - OK, 0 - FAIL.                            |
| pg_probackup_wal_bytes            | Gauge    | WAL size in bytes                                                     |

## Usage

Build app

```bash
go build -o pg_probackup_exporter main.go
```

Copy service and app from repo to yor host and start service.

In service you should write your path to backup pg_probackup directory and path to pg_probackup util:

Example isntallation.

```bash
sudo cp pg_probackup_exporter.service /etc/systemd/system/pg_probackup_exporter.service
sudo cp pg_probackup_exporter /usr/bin/pg_probackup_exporter /usr/bin/pg_probackup_exporter
sudo chown postgres:postgres /usr/bin/pg_probackup_exporter
sudo chmod +x 
sudo systemctl daemon-reload
sudo systemctl enalbe pg_probackup_exporter.service
sudo systemctl start pg_probackup_exporter.service
```

On Prometheus server:
```yaml
  - job_name: 'pg_probackup_metrics'
    scrape_interval: 60s
    scrape_timeout: 5s
    static_configs:
    - targets:
      - your_host:9231
```
