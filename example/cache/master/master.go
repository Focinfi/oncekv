package main

import (
	"os"

	"github.com/Focinfi/oncekv/cache/master"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "node"
	app.Action = func(c *cli.Context) error {
		master := master.New(c.Args().Get(0))
		master.Start()
		return nil
	}

	app.Run(os.Args)
}
