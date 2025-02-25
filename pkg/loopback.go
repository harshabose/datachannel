package data

import (
	"context"
	"fmt"
	"github.com/pion/webrtc/v4"
	"net"
	"os/exec"
)

type LoopBack struct {
	dataChannel  *webrtc.DataChannel
	bindPortConn *net.UDPConn
	remotePort   *net.UDPAddr
	mavp2p       *exec.Cmd
	ctx          context.Context
}

func CreateLoopBack(ctx context.Context, options ...LoopBackOption) (*LoopBack, error) {
	loopback := &LoopBack{
		ctx: ctx,
	}

	for _, option := range options {
		if err := option(loopback); err != nil {
			return nil, err
		}
	}

	if loopback.bindPortConn == nil {
		if err := WithRandomBindPort(loopback); err != nil {
			return nil, err
		}
	}

	return loopback, nil
}

func (loopback *LoopBack) start() {
	if err := loopback.mavp2p.Start(); err != nil {
		fmt.Printf("Error starting mavp2p loopback: %v... Skipping\n", err)
	}
	go loopback.loop()
}

func (loopback *LoopBack) loop() {
	var (
		buffer []byte
		nRead  = 0
	)

	defer loopback.Close()

	for {
		select {
		case <-loopback.ctx.Done():
			return
		default:

			if buffer, nRead = loopback.readMessageFromUDPPort(); nRead > 0 && nRead < 1025 {
				loopback.sendMessageThroughDataChannel(buffer[:nRead])
			}
		}
	}
}

func (loopback *LoopBack) Close() error {
	if loopback.bindPortConn != nil {
		if err := loopback.bindPortConn.Close(); err != nil {
			return err
		}
	}

	if loopback.mavp2p != nil && loopback.mavp2p.Process != nil {
		if err := loopback.mavp2p.Process.Kill(); err != nil {
			return err
		}
	}

	return nil
}

func (loopback *LoopBack) Send(message []byte) error {
	var (
		bytesWritten       = 0
		err          error = nil
	)

	if loopback.bindPortConn == nil {
		return fmt.Errorf("bind port not yet set. Skipping message")
	}
	if loopback.remotePort == nil {
		return fmt.Errorf("loopback port not yet discovered. Skipping message")
	}
	if bytesWritten, err = loopback.bindPortConn.WriteToUDP(message, loopback.remotePort); err == nil {
		if bytesWritten != len(message) {
			err = fmt.Errorf("written bytes (%d) != message length (%d)", bytesWritten, len(message))
		}
	}
	return err
}

func (loopback *LoopBack) readMessageFromUDPPort() ([]byte, int) {
	var (
		buffer     []byte       = make([]byte, 1024)
		nRead                   = 0
		senderAddr *net.UDPAddr = nil
		err        error        = nil
	)

	if nRead, senderAddr, err = loopback.bindPortConn.ReadFromUDP(buffer); err != nil {
		fmt.Println("Error while reading message from bind port" + err.Error())
		return nil, 0
	}

	if loopback.remotePort == nil {
		loopback.remotePort = &net.UDPAddr{IP: senderAddr.IP, Port: senderAddr.Port}
		fmt.Println("Found sender port to bind port")
	}

	if senderAddr != nil && senderAddr.Port != loopback.remotePort.Port {
		fmt.Println(fmt.Sprintf("expected port %d but got %d", loopback.remotePort.Port, senderAddr.Port))
	}

	return buffer, nRead
}

func (loopback *LoopBack) sendMessageThroughDataChannel(message []byte) {
	var err error = nil

	if loopback.dataChannel == nil {
		fmt.Println("datachannel not yet set")
		return
	}
	if loopback.dataChannel.ReadyState() == webrtc.DataChannelStateOpen {
		if err = loopback.dataChannel.Send(message); err != nil {
			fmt.Println("failed to send data: " + err.Error())
		}
		return
	}
}
