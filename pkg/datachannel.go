package data

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/pion/webrtc/v4"
)

type DataChannel struct {
	label       string
	datachannel *webrtc.DataChannel
	loopback    *LoopBack
	ctx         context.Context
}

func CreateDataChannel(ctx context.Context, label string, peerConnection *webrtc.PeerConnection, loopback *LoopBack) (*DataChannel, error) {
	datachannel := &DataChannel{
		label:       label,
		datachannel: nil,
		loopback:    loopback,
		ctx:         ctx,
	}
	var (
		dataChannelNegotiated = true
		dataChannelProtocol   = "binary"
		dataChannelOrdered    = true
		dataChannelID         = uint16(1) // Add explicit ID
		dataChannelInit       = webrtc.DataChannelInit{
			Negotiated: &dataChannelNegotiated,
			Protocol:   &dataChannelProtocol,
			Ordered:    &dataChannelOrdered,
			ID:         &dataChannelID,
		}
		err error
	)

	if datachannel.datachannel, err = peerConnection.CreateDataChannel(label, &dataChannelInit); err != nil {
		return nil, err
	}

	loopback.dataChannel = datachannel.datachannel
	loopback.start()

	return datachannel.onOpen().onClose().onMessage(), nil
}

func (dataChannel *DataChannel) GetLabel() string {
	return dataChannel.label
}

func (dataChannel *DataChannel) GetBindPort() int {
	return dataChannel.loopback.bindPortConn.LocalAddr().(*net.UDPAddr).Port
}

func (dataChannel *DataChannel) Close() error {
	dataChannel.loopback.Close()
	if err := dataChannel.datachannel.Close(); err != nil {
		return err
	}

	return nil
}

func (dataChannel *DataChannel) onOpen() *DataChannel {
	dataChannel.datachannel.OnOpen(func() {
		fmt.Printf("dataChannel Open with Label: %s\n", dataChannel.datachannel.Label())
	})
	return dataChannel
}

func (dataChannel *DataChannel) onClose() *DataChannel {
	dataChannel.datachannel.OnClose(func() {
		fmt.Printf("dataChannel Closed with Label: %s\n", dataChannel.datachannel.Label())
	})
	return dataChannel
}

func (dataChannel *DataChannel) onMessage() *DataChannel {
	dataChannel.datachannel.OnMessage(func(message webrtc.DataChannelMessage) {
		if err := dataChannel.loopback.Send(message.Data); err != nil {
			fmt.Println("Error sending data: " + err.Error())
		}
	})
	return dataChannel
}

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

type DataChannels struct {
	datachannel map[string]*DataChannel
	ctx         context.Context
}

func CreateDataChannels(ctx context.Context) (*DataChannels, error) {
	return &DataChannels{
		datachannel: map[string]*DataChannel{},
		ctx:         ctx,
	}, nil
}

func (dataChannels *DataChannels) CreateDataChannel(label string, peerConnection *webrtc.PeerConnection, options ...LoopBackOption) (*DataChannel, error) {
	var (
		loopback *LoopBack
		err      error
	)

	if loopback, err = CreateLoopBack(dataChannels.ctx, options...); err != nil {
		return nil, err
	}
	if dataChannels.datachannel[label], err = CreateDataChannel(dataChannels.ctx, label, peerConnection, loopback); err != nil {
		return nil, err
	}

	return dataChannels.datachannel[label], nil
}

func (dataChannels *DataChannels) GetDataChannel(label string) (*DataChannel, error) {
	dataChannel, exists := dataChannels.datachannel[label]
	if !exists {
		return nil, errors.New("datachannel does not exists")
	}
	return dataChannel, nil
}

func (dataChannels *DataChannels) Close(label string) (err error) {
	if err = dataChannels.datachannel[label].Close(); err == nil {
		return nil
	}
	delete(dataChannels.datachannel, label)
	return err
}
