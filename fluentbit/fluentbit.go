package fluentbit

import (
	"bytes"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"
	"text/template"
	"time"

	"github.com/hashicorp/nomad/api"

	"github.com/dmaes/nomad-logger/nomad"
	"github.com/dmaes/nomad-logger/util"
)

type Fluentbit struct {
	Nomad         *nomad.Nomad
	ConfFile      string
	TagPrefix     string
	Parser        string
	ReloadCmd     string
	CheckInterval int64
}

//go:embed fluentbit-conf.gotmpl
var FluentbitConfTmpl string

func (f *Fluentbit) Run(m *util.Metrics) {
	for {
		time.Sleep(1 * time.Second)
		allocs, err := f.Nomad.Allocs()
		if err != nil {
			slog.Error(err.Error())
		}

		m.Allocs.Set(float64(len(allocs)))
		updateErr := f.UpdateConf(allocs)
		if updateErr != nil {
			slog.Error(updateErr.Error())
		}
	}
}

func (f *Fluentbit) UpdateConf(Allocs []*api.Allocation) error {
	config := ""

	for _, alloc := range Allocs {
		allocConfig, allocErr := f.AllocToConfig(alloc)
		if allocErr != nil {
			return allocErr
		}
		config += allocConfig
	}

	writeErr := util.WriteConfig(config, f.ConfFile, f.ReloadCmd)
	if writeErr != nil {
		return writeErr
	}
	return nil
}

func (f *Fluentbit) AllocToConfig(Alloc *api.Allocation) (string, error) {
	tasks, err := nomad.AllocTasks(Alloc)
	if err != nil {
		return "", err
	}

	config := ""

	for _, task := range tasks {
		stdOutConfig, stdOutErr := f.AllocTaskStreamToConfig(Alloc, task, "stdout")
		if stdOutErr != nil {
			return "", stdOutErr
		}
		config += stdOutConfig
		stdErrConfig, stdErrErr := f.AllocTaskStreamToConfig(Alloc, task, "stderr")
		if stdErrErr != nil {
			return "", stdErrErr
		}
		config += stdErrConfig
	}

	return config, nil
}

func (f *Fluentbit) AllocTaskStreamToConfig(Alloc *api.Allocation, Task *api.Task, Stream string) (string, error) {
	tagPrefix := f.Nomad.TaskMetaGet(*Task, "fluentbit.tag-prefix", f.TagPrefix)
	tag := fmt.Sprintf("%s.%s.%s.%s", tagPrefix, Alloc.ID, Task.Name, Stream)

	path := fmt.Sprintf("%s/%s/alloc/logs/%s.%s.[0-9]*", f.Nomad.AllocsDir, Alloc.ID, Task.Name, Stream)

	parser := f.Nomad.TaskMetaGet(*Task, "fluentbit.parser", f.Parser)

	filterParsersStr := f.Nomad.TaskMetaGet(*Task, "fluentbit.filter-parsers", "")
	filterParsers := make([]*FluentbitFilterParser, 0)
	if filterParsersStr != "" {
		for _, fp := range strings.Split(filterParsersStr, ",") {
			if fp != "" {
				fpSplit := strings.Split(fp, ":")
				filterParsers = append(filterParsers, &FluentbitFilterParser{
					Key:    fpSplit[0],
					Parser: fpSplit[1],
				})
			}
		}
	}

	fluentbitConfig := &FluentbitConfig{
		Tag:            tag,
		Path:           path,
		Parser:         parser,
		FilterParsers:  filterParsers,
		NomadNamespace: Alloc.Namespace,
		NomadJob:       Alloc.JobID,
		NomadTaskGroup: Alloc.TaskGroup,
		NomadTask:      Task.Name,
		NomadAllocID:   Alloc.ID,
		NomadAllocName: Alloc.Name,
		NomadNodeID:    f.Nomad.NodeID,
		NomadLogStream: Stream,
	}

	tpl := template.Must(template.New("fluentbit-conf").Parse(FluentbitConfTmpl))
	var tplBuffer bytes.Buffer
	err := tpl.Execute(&tplBuffer, fluentbitConfig)
	if err != nil {
		return "", err
	}

	return tplBuffer.String(), nil
}

type FluentbitConfig struct {
	Tag            string
	Path           string
	Parser         string
	FilterParsers  []*FluentbitFilterParser
	NomadNamespace string
	NomadJob       string
	NomadTaskGroup string
	NomadTask      string
	NomadAllocID   string
	NomadAllocName string
	NomadNodeID    string
	NomadLogStream string
}

type FluentbitFilterParser struct {
	Key    string
	Parser string
}
