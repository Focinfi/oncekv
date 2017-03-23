package main

import (
	"os"

	"github.com/Focinfi/oncekv/db/node/service"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "node"
	app.Action = func(c *cli.Context) error {
		master := service.New(c.Args().Get(0), c.Args().Get(1), c.Args().Get(2))
		master.Start()
		return nil
	}

	app.Run(os.Args)
}
