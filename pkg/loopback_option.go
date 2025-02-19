package data

import "net"

type LoopBackOption = func(*LoopBack) error

func WithBindPort(loopback *LoopBack) error {
	var err error
	if loopback.udpListener, err = net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: DefaultBindPort}); err != nil {
		return err
	}
	return nil
}

func WithLoopBackPort(loopback *LoopBack) error {
	loopback.loopBackPort = &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: DefaultLoopBackPort}
	return nil
}

func WithMAVP2P(loopback *LoopBack) error {
	return nil
}
