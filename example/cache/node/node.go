package main

import (
	"os"

	"github.com/Focinfi/oncekv/cache/node"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "node"
	app.Action = func(c *cli.Context) error {
		node := node.New(c.Args().Get(0), c.Args().Get(1), c.Args().Get(2))
		node.Start()
		return nil
	}

	app.Run(os.Args)
}
