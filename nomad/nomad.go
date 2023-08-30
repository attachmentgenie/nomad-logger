package nomad

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/hashicorp/nomad/api"
)

type Nomad struct {
	Client     *api.Client
	Address    string
	AllocsDir  string
	NodeID     string
	MetaPrefix string
}

func (n *Nomad) NewClient() error {
	config := *api.DefaultConfig()
	config.Address = n.Address
	client, err := api.NewClient(&config)
	if err != nil {
		return err
	}

	n.Client = client
	return nil
}

func (n *Nomad) Allocs() ([]*api.Allocation, error) {
	allocs, _, err := n.Client.Nodes().Allocations(n.NodeID, nil)
	if err != nil {
		return nil, err
	}
	return allocs, nil
}

func (n *Nomad) TaskMeta(Task api.Task) map[string]string {
	meta := make(map[string]string)

	regex, _ := regexp.Compile(fmt.Sprintf("^(%s)\\.", n.MetaPrefix))
	for key, value := range Task.Meta {
		if regex.MatchString(key) {
			strippedKey := regex.ReplaceAllString(key, "")
			meta[strippedKey] = value
		}
	}

	return meta
}

func (n *Nomad) TaskMetaGet(Task api.Task, Key string, Default string) string {
	meta := n.TaskMeta(Task)

	value, exists := meta[Key]

	if exists {
		return value
	}

	return Default
}

func AllocTasks(Alloc *api.Allocation) ([]*api.Task, error) {
	for _, group := range Alloc.Job.TaskGroups {
		if *group.Name == Alloc.TaskGroup {
			return group.Tasks, nil
		}
	}

	return nil, errors.New("could not find Tasks for Allocation")
}
