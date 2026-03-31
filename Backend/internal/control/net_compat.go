package control

import "net"

func netListen(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}
