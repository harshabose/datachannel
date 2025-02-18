package pkg

import (
	"context"
	"fmt"

	"github.com/pion/webrtc/v4"
)

type DataChannels map[string]*DataChannel

type DataChannel struct {
	datachannel *webrtc.DataChannel
	loopback    *LoopBack
	ctx         context.Context
}

func CreateDataChannel(ctx context.Context, label string, peerConnection *webrtc.PeerConnection, loopback *LoopBack) (*DataChannel, error) {
	var (
		datachannel           *webrtc.DataChannel
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

	if datachannel, err = peerConnection.CreateDataChannel(label, &dataChannelInit); err != nil {
		return nil, err
	}

	if err = loopback.AttachDataChannel(datachannel); err != nil {
		return nil, err
	}

	return &DataChannel{
		datachannel: datachannel,
		loopback:    loopback,
		ctx:         ctx,
	}, nil
}

func (dataChannel *DataChannel) Label() string {
	return dataChannel.datachannel.Label()
}

func (dataChannel *DataChannel) Send(message []byte) error {
	return dataChannel.datachannel.Send(message)
}

func (dataChannel *DataChannel) Close() (err error) {
	if err = dataChannel.datachannel.Close(); err == nil {
		return nil
	}
	return err
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
