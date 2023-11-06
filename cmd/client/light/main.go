/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package main

import (
	"log"
	"os"

	"github.com/elcn233/go-scdo/cmd/client/cmd"
)

func main() {
	app := cmd.NewApp(false)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
