package main

import (
	"os"

	"github.com/Focinfi/oncekv/groupcache/node"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "node"
	app.Action = func(c *cli.Context) error {
		node := node.New(c.Args().Get(0), c.Args().Get(1))
		node.Start()
		return nil
	}

	app.Run(os.Args)
}
