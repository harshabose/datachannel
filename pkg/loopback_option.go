package data

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
)

type LoopBackOption = func(*LoopBack) error

func WithRandomBindPort(loopback *LoopBack) error {
	var err error
	if loopback.bindPortConn, err = net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}); err != nil {
		return err
	}

	return nil
}

func WithBindPort(port int) LoopBackOption {
	return func(loopback *LoopBack) error {
		var err error
		if loopback.bindPortConn, err = net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}); err != nil {
			return err
		}
		return nil
	}
}

func WithLoopBackPort(port int) LoopBackOption {
	return func(loopback *LoopBack) error {
		loopback.remotePort = &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}
		return nil
	}
}

func WithMAVP2P(path string, serial string) LoopBackOption {
	return func(loopback *LoopBack) error {
		if loopback.bindPortConn == nil {
			return errors.New("bindPortConn not initialized, call WithBindPort or WithRandomBindPort first")
		}

		port := loopback.bindPortConn.LocalAddr().(*net.UDPAddr).Port
		ser := fmt.Sprintf("serial:%s", serial)
		addr := fmt.Sprintf("udpc:%s:%d", "127.0.0.1", port)
		cmd := exec.Command(path, ser, addr)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		loopback.mavp2p = cmd

		return nil
	}
}
