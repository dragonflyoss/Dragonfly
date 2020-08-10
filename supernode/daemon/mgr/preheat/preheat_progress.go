package preheat

import (
	"os/exec"
)

type PreheatProgress struct {
	output string
	cmd *exec.Cmd
}

func NewPreheatProgress(output string, cmd *exec.Cmd) *PreheatProgress {
	p := &PreheatProgress{
		output: output,
		cmd: cmd,
	}
	return p
}