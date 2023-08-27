package services

import (
	"autoshell/core/entities"
	"autoshell/core/ports"
	"errors"
	"fmt"
	"strings"
)

type runner struct{}

func NewRunner() ports.Runner {
	return &runner{}
}

func (r *runner) Run(config *entities.Config, workflowName string) error {
	cmdsStr, ok := config.Workflows[workflowName]
	if !ok {
		return errors.New("workflow not found")
	}
	cmds := strings.Split(cmdsStr, "\n")
	for _, cmd := range cmds {
		fmt.Println(cmd)
	}
	return nil
}
