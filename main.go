package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Определение метрик как GaugeVec для хранения статуса бэкапа, размера данных, размера WAL и проверки целостности.
var (
	pgProbackupStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pg_probackup_status",
			Help: "Status of current backup 1 - OK, 0 - FAIL.",
		},
		[]string{"instance", "backup_mode", "backup_id"},
	)
	pgProbackupSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pg_probackup_size_bytes",
			Help: "Size in bytes",
		},
		[]string{"instance", "backup_mode", "backup_id"},
	)
	pgProbackupWalSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pg_probackup_wal_bytes",
			Help: "WAL size in bytes",
		},
		[]string{"instance", "backup_mode", "backup_id"},
	)
	pgProbackupIntegrityCheck = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pg_probackup_integrity_check",
			Help: "Integrity check status 1 - OK, 0 - FAIL",
		},
		[]string{"instance", "backup_mode", "backup_id"},
	)
)

// init - инициализация метрик и их регистрация в Prometheus.
func init() {
	prometheus.MustRegister(pgProbackupStatus)
	prometheus.MustRegister(pgProbackupSize)
	prometheus.MustRegister(pgProbackupWalSize)
	prometheus.MustRegister(pgProbackupIntegrityCheck)
}

// getPgProbackupStatus - выполнение команды pg_probackup и получение данных о статусе бэкапов.
func getPgProbackupStatus(backupPath string, pgProBackupVersion string) ([]map[string]interface{}, error) {
	cmd := exec.Command(pgProBackupVersion, "show", "--backup-path", backupPath, "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running pg_probackup: %v", err)
	}

	var backups []map[string]interface{}
	if err := json.Unmarshal(output, &backups); err != nil {
		return nil, fmt.Errorf("error unmarshaling json: %v", err)
	}
	return backups, nil
}

// parseBackupStatus - обработка данных о статусе бэкапов и обновление метрик.
func parseBackupStatus(data []map[string]interface{}) {
	for _, instance := range data {
		instanceName := instance["instance"].(string)
		backups := instance["backups"].([]interface{})

		for _, b := range backups {
			backup := b.(map[string]interface{})
			backupID := backup["id"].(string)
			status := backup["status"].(string)
			backupMode := backup["backup-mode"].(string)
			dataBytes := backup["data-bytes"].(float64)
			walBytes := backup["wal-bytes"].(float64)

			// Установка значения метрики статуса бэкапа в 1 - успешный, 0 - неуспешный.
			pgProbackupStatus.WithLabelValues(instanceName, backupMode, backupID).Set(float64(0))
			if status == "OK" {
				pgProbackupStatus.WithLabelValues(instanceName, backupMode, backupID).Set(float64(1))
			}

			// Установка значения метрики размера данных бэкапа.
			pgProbackupSize.WithLabelValues(instanceName, backupMode, backupID).Set(dataBytes)

			// Установка значения метрики размера WAL.
			pgProbackupWalSize.WithLabelValues(instanceName, backupMode, backupID).Set(walBytes)

			// Установка значения метрики проверки целостности бэкапа в 1 - успешная, 0 - неуспешная.
			pgProbackupIntegrityCheck.WithLabelValues(instanceName, backupMode, backupID).Set(float64(0))
			if status == "OK" {
				pgProbackupIntegrityCheck.WithLabelValues(instanceName, backupMode, backupID).Set(float64(1))
			}
		}
	}
}

func main() {

	var (
		backupPath         string // Путь к директории с бэкапами pg_probackup
		pgProBackupVersion string // Имя исполняемого файла pg_probackup, f.e /usr/bin/pg_probackup-14
		showHelp           bool
	)

	flag.StringVar(&backupPath, "data-dir", "", "Path to the directory with pg_probackup backups")
	flag.StringVar(&pgProBackupVersion, "pg_probackup_path", "", "Path to the pg_probackup executable")
	flag.BoolVar(&showHelp, "help", false, "Show help message")

	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if backupPath == "" || pgProBackupVersion == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Запуск HTTP сервера на порту 9231
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		fmt.Println("Starting server on port 9231")
		if err := http.ListenAndServe(":9231", nil); err != nil {
			log.Fatalf("Error starting HTTP server: %v", err)
		}
	}()

	// Бесконечный цикл для обновления метрик
	for {
		backups, err := getPgProbackupStatus(backupPath, pgProBackupVersion)
		if err != nil {
			log.Printf("Error getting pg_probackup status: %v", err)
		} else {
			parseBackupStatus(backups)
		}
		// Пауза на 60 секунд перед следующим обновлением метрик.
		time.Sleep(60 * time.Second)
	}
}
