package pkg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/pion/webrtc/v4"
)

type LoopBack struct {
	dataChannel  *webrtc.DataChannel
	udpListener  *net.UDPConn
	loopBackPort *net.UDPAddr
	ctx          context.Context
}

func CreateLoopBack(ctx context.Context, options ...LoopBackOption) (*LoopBack, error) {
	loopBack := &LoopBack{
		ctx: ctx,
	}

	for _, option := range options {
		if err := option(loopBack); err != nil {
			return nil, err
		}
	}

	return loopBack, nil
}

func (loopback *LoopBack) AttachDataChannel(datachannel *webrtc.DataChannel) error {
	if loopback.dataChannel == nil {
		loopback.dataChannel = datachannel
		return nil
	}
	return errors.New("datachannel already attached")
}

func (loopback *LoopBack) Start() {
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

			if loopback.udpListener == nil {
				fmt.Println("Bind port not yet detected. Sleeping for 0.1 second...")
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if buffer, nRead = loopback.readMessageFromUDPPort(); nRead > 0 && nRead < 1025 {
				loopback.sendMessageThroughDataChannel(buffer[:nRead])
			}
		}
	}
}

func (loopback *LoopBack) Close() {
	if loopback.udpListener != nil {
		loopback.udpListener.Close()
	}
}

func (loopback *LoopBack) Send(message []byte) error {
	var (
		bytesWritten       = 0
		err          error = nil
	)

	if loopback.udpListener == nil {
		return fmt.Errorf("bind port not yet set. Skipping message")
	}
	if loopback.loopBackPort == nil {
		return fmt.Errorf("loopback port not yet discovered. Skipping message")
	}
	if bytesWritten, err = loopback.udpListener.WriteToUDP(message, loopback.loopBackPort); err == nil {
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

	if nRead, senderAddr, err = loopback.udpListener.ReadFromUDP(buffer); err != nil {
		fmt.Println("Error while reading message from bind port" + err.Error())
		return nil, 0
	}

	if loopback.loopBackPort == nil {
		loopback.loopBackPort = &net.UDPAddr{IP: senderAddr.IP, Port: senderAddr.Port}
		fmt.Println("Found sender port to bind port")
	}

	if senderAddr != nil && senderAddr.Port != loopback.loopBackPort.Port {
		fmt.Println(fmt.Sprintf("expected port %d but got %d", loopback.loopBackPort.Port, senderAddr.Port))
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
