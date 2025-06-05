package data

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"

	"github.com/pion/webrtc/v4"
)

type LoopBack struct {
	dataChannel  *webrtc.DataChannel
	bindPortConn *net.UDPConn
	remotePort   *net.UDPAddr
	mavp2p       *exec.Cmd
	mavproxy     *exec.Cmd
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

func (l *LoopBack) start() {
	if l.mavp2p != nil {
		if err := l.mavp2p.Start(); err != nil {
			fmt.Printf("Error starting mavp2p l: %v... Skipping\n", err)
		}
	}

	if l.mavproxy != nil {
		if err := l.mavproxy.Start(); err != nil {
			fmt.Printf("error starting mavproxy: %v... skipping\n", err.Error())
		}
	}

	go l.loop()
}

func (l *LoopBack) loop() {
	var (
		buffer []byte
		nRead  = 0
	)

	defer l.Close()

	for {
		select {
		case <-l.ctx.Done():
			return
		default:
			if buffer, nRead = l.readMessageFromUDPPort(); nRead > 0 && nRead < 1025 {
				if err := l.sendMessageThroughDataChannel(buffer[:nRead]); err != nil {
					fmt.Println("error in l; err:", err.Error())
					continue
				}
			}
		}
	}
}

func (l *LoopBack) Close() {
	if l.bindPortConn != nil {
		if err := l.bindPortConn.Close(); err != nil {
			fmt.Println("error while closing the l; err:", err.Error())
		}
	}

	if l.mavp2p != nil && l.mavp2p.Process != nil {
		if err := l.mavp2p.Process.Kill(); err != nil {
			fmt.Println("error while closing the l; err:", err.Error())
		}
	}
}

func (l *LoopBack) Send(message []byte) error {
	if l.bindPortConn == nil {
		return fmt.Errorf("bind port not yet set. Skipping message")
	}
	if l.remotePort == nil {
		return fmt.Errorf("l port not yet discovered. Skipping message")
	}
	bytesWritten, err := l.bindPortConn.WriteToUDP(message, l.remotePort)
	if bytesWritten != len(message) {
		return fmt.Errorf("written bytes (%d) != message length (%d)", bytesWritten, len(message))
	}
	if err != nil {
		return err
	}

	return nil
}

func (l *LoopBack) readMessageFromUDPPort() ([]byte, int) {
	buffer := make([]byte, 1024)

	nRead, senderAddr, err := l.bindPortConn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println("Error while reading message from bind port" + err.Error())
		return nil, 0
	}

	if l.remotePort == nil {
		l.remotePort = &net.UDPAddr{IP: senderAddr.IP, Port: senderAddr.Port}
		fmt.Println("Found sender port to bind port")
	}

	if senderAddr != nil && senderAddr.Port != l.remotePort.Port {
		fmt.Println(fmt.Sprintf("expected port %d but got %d", l.remotePort.Port, senderAddr.Port))
	}

	return buffer, nRead
}

func (l *LoopBack) sendMessageThroughDataChannel(message []byte) error {
	if l.dataChannel == nil {
		return errors.New("datachannel not yet set")
	}

	if l.dataChannel.ReadyState() == webrtc.DataChannelStateOpen {
		err := l.dataChannel.Send(message)
		if err != nil {
			return fmt.Errorf("failed to send data through data channel; err: %s", err.Error())
		}

		return nil
	}

	return errors.New("datachannel not in ready mode")
}
