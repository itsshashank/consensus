package app

import "github.com/itsshashank/consensus/internal/core"

func StartCluster(nodeCount int, net core.Network) []*core.Node {
	var nodes []*core.Node
	for i := 0; i < nodeCount; i++ {
		peers := makePeers(i, nodeCount)
		n := core.NewNode(i, peers, net)
		go n.Run()
		nodes = append(nodes, n)
	}
	return nodes
}

func makePeers(self, total int) []int {
	var peers []int
	for i := range total {
		if i != self {
			peers = append(peers, i)
		}
	}
	return peers
}
