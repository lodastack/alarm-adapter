package main

import (
	"os"

	"github.com/lodastack/alarm-adapter/command"
	"github.com/lodastack/alarm-adapter/config"

	"github.com/oiooj/cli"
)

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = config.AppName
	app.Usage = config.Usage
	app.Version = config.Version
	app.Author = config.Author
	app.Email = config.Email

	app.Commands = []cli.Command{
		command.CmdStart,
		command.CmdStop,
	}

	app.Flags = append(app.Flags, []cli.Flag{}...)
	app.Run(os.Args)
}
