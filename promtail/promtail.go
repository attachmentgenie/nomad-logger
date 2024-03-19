package promtail

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/attachmentgenie/nomad-logger/nomad"
	"github.com/attachmentgenie/nomad-logger/util"

	"github.com/hashicorp/nomad/api"
	"gopkg.in/yaml.v3"
)

type Promtail struct {
	Nomad         *nomad.Nomad
	TargetsFile   string
	CheckInterval int64
}

func (p *Promtail) Run(m *util.Metrics) {
	for {
		time.Sleep(1 * time.Second)
		allocs, err := p.Nomad.Allocs()
		if err != nil {
			slog.Error(err.Error())
		}

		m.Allocs.Set(float64(len(allocs)))
		updateErr := p.UpdatePromtailTargets(allocs)
		if updateErr != nil {
			slog.Error(updateErr.Error())
		}
	}
}

func (p *Promtail) UpdatePromtailTargets(Allocs []*api.Allocation) error {
	config := []*ScrapeConfig{}

	for _, alloc := range Allocs {
		ScrapeConfigs, err := p.AllocToScrapeConfigs(alloc)
		if err != nil {
			slog.Error(err.Error())
		}
		config = append(config, ScrapeConfigs...)
	}

	yamlBytes, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}
	yamlString := string(yamlBytes)

	writeErr := util.WriteConfig(yamlString, p.TargetsFile, "")
	if writeErr != nil {
		return writeErr
	}

	return nil
}

func (p *Promtail) AllocToScrapeConfigs(Alloc *api.Allocation) ([]*ScrapeConfig, error) {
	configs := []*ScrapeConfig{}

	tasks, err := nomad.AllocTasks(Alloc)
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		configs = append(configs, p.AllocTaskStreamToScrapeConfig(Alloc, task, "stdout"))
		configs = append(configs, p.AllocTaskStreamToScrapeConfig(Alloc, task, "stderr"))
	}

	return configs, nil
}

func (p *Promtail) AllocTaskStreamToScrapeConfig(Alloc *api.Allocation, Task *api.Task, Stream string) *ScrapeConfig {
	config := &ScrapeConfig{
		Targets: []string{"localhost"},
		Labels: map[string]string{
			"nomad_namespace":  Alloc.Namespace,
			"nomad_job":        Alloc.JobID,
			"nomad_task_group": Alloc.TaskGroup,
			"nomad_task":       Task.Name,
			"nomad_alloc_id":   Alloc.ID,
			"nomad_alloc_name": Alloc.Name,
			"nomad_node_id":    p.Nomad.NodeID,
			"nomad_log_stream": Stream,

			"__path__": fmt.Sprintf("%s/%s/alloc/logs/%s.%s.[0-9]*", p.Nomad.AllocsDir, Alloc.ID, Task.Name, Stream),
		},
	}
	return config
}

type ScrapeConfig struct {
	Targets []string          `yaml:"targets"`
	Labels  map[string]string `yaml:"labels"`
}
