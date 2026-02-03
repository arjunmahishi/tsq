package main

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/urfave/cli/v3"
)

//go:embed SKILL.md
var skillText string

func skillCommand() *cli.Command {
	return &cli.Command{
		Name:  "skill",
		Usage: "print SKILL.md for agent configs",
		Description: "Print the Anthropic SKILLS framework documentation for tsq.\n" +
			"Use this to integrate tsq into agent configurations that support the SKILLS format.\n\n" +
			"Examples:\n" +
			"  tsq skill                    # print full SKILL.md\n" +
			"  tsq skill > my-agent.skill   # save to file",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Print(skillText)
			return nil
		},
	}
}
