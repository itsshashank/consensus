package main

import (
	"time"

	"github.com/itsshashank/consensus/internal/adapter/network"
	"github.com/itsshashank/consensus/internal/app"
)

func main() {
	net := network.New()
	nodes := app.StartCluster(5, net)

	time.Sleep(400 * time.Millisecond)
	go func() {
		for _, n := range nodes {
			n.ProposeCommand("Set x = 1")
		}
	}()
	_ = nodes
	select {}
}
