package util

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"os"
	"os/exec"
)

type Metrics struct {
	Allocs prometheus.Gauge
}

func NewMetrics() *Metrics {
	m := &Metrics{
		Allocs: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "nomad_logger",
			Name:      "allocs_processed",
		}),
	}
	prometheus.MustRegister(m.Allocs)
	return m
}

func WriteConfig(Config string, File string, ReloadCmd string) error {
	oldConfig := ""
	oldConfBytes, err := os.ReadFile(File)
	if err != nil && err.Error() != fmt.Sprintf("open %s: no such file or directory", File) {
		slog.Error(err.Error())
		return err
	} else if err == nil {
		oldConfig = string(oldConfBytes)
	}

	if oldConfig == Config {
		return nil
	}

	slog.Info("Updating config")
	writeErr := os.WriteFile(File, []byte(Config), 0644)
	if writeErr != nil {
		return writeErr
	}

	if ReloadCmd == "" {
		return nil
	}

	slog.Info("Executing ReloadCmd")
	out, cmdErr := exec.Command("/bin/sh", "-c", ReloadCmd).CombinedOutput()
	slog.Info(string(out))
	if cmdErr != nil {
		slog.Error(cmdErr.Error())
		return cmdErr
	}
	return nil
}
