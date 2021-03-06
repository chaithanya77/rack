package main

import (
	"fmt"
	"strings"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("instances", "list instances", Instances, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	CLI.Command("instances keyroll", "roll ssh key on instances", InstancesKeyroll, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Validate: stdcli.Args(0),
	})

	CLI.Command("instances ssh", "run a shell on an instance", InstancesSsh, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.ArgsMin(1),
	})
}

func Instances(c *stdcli.Context) error {
	is, err := provider(c).InstanceList()
	if err != nil {
		return err
	}

	t := c.Table("ID", "STATUS", "STARTED", "PS", "CPU", "MEM", "PUBLIC", "PRIVATE")

	for _, i := range is {
		t.AddRow(i.Id, i.Status, helpers.Ago(i.Started), fmt.Sprintf("%d", i.Processes), helpers.Percent(i.Cpu), helpers.Percent(i.Memory), i.PublicIp, i.PrivateIp)
	}

	return t.Print()
}

func InstancesKeyroll(c *stdcli.Context) error {
	c.Startf("Rolling instance key")

	if err := provider(c).InstanceKeyroll(); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForRackRunning(c); err != nil {
			return err
		}
	}

	return c.OK()
}

func InstancesSsh(c *stdcli.Context) error {
	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	w, h, err := c.TerminalSize()
	if err != nil {
		return err
	}

	opts := structs.InstanceShellOptions{
		Height: options.Int(h),
		Width:  options.Int(w),
	}

	command := strings.Join(c.Args[1:], " ")

	if command != "" {
		opts.Command = options.String(command)
	}

	if s.Version <= "20180708231844" {
		opts.Command = options.String(strings.Join(c.Args[1:], " "))
		opts.Height = options.Int(h)
		opts.Width = options.Int(w)

		code, err := provider(c).InstanceShellClassic(c.Arg(0), c, opts)
		if err != nil {
			return err
		}

		return stdcli.Exit(code)
	}

	if err := c.TerminalRaw(); err != nil {
		return err
	}

	defer c.TerminalRestore()

	code, err := provider(c).InstanceShell(c.Arg(0), c, opts)
	if err != nil {
		return err
	}

	return stdcli.Exit(code)
}
