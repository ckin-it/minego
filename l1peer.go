package main

import (
	"fmt"
	"log"
	"time"

	"github.com/pion/webrtc"
)

type l1peer struct {
	Name      string
	pc        *webrtc.PeerConnection
	dc        *webrtc.DataChannel
	rtcConfig *webrtc.Configuration
}

func strPtr(a string) *string {
	return &a
}

// newL1Peer is the creator
func newL1Peer(name string) (p *l1peer) {
	p = new(l1peer)
	p.Name = name
	p.rtcConfig = &webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	pc, err := webrtc.NewPeerConnection(*p.rtcConfig)
	pc.OnICEGatheringStateChange(func(gather webrtc.ICEGathererState) {
		log.Println(gather)
	})
	checkPanic(err)

	pc.OnICECandidate(func(ice *webrtc.ICECandidate) {
		desc := pc.LocalDescription()
		log.Println("ONICDECANDIDATE", p.Name, "OnIceCandidate", desc)
		pc.SetLocalDescription(*desc)
		//backchannel <- desc
	})

	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Println(p.Name, "ICE Connection State has changed:", connectionState.String())
	})

	// Register data channel creation handling
	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Println("New DataChannel:", d.Label(), d.ID())
		p.dc = d

		// Register channel opening handling
		d.OnOpen(func() {
			log.Println("Data channel ", d.Label(), d.ID(), "open. DO SOMETHING")

			for range time.NewTicker(5 * time.Second).C {
				message := "AAAA"
				fmt.Printf("Sending '%s'\n", message)

				// Send the message as text
				sendErr := d.SendText(message)
				checkPanic(sendErr)
			}
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
		})
	})
	_true := true
	opts := webrtc.DataChannelInit{
		Negotiated: &_true,
	}
	pc.CreateDataChannel(p.Name, &opts)
	p.pc = pc

	return p
}
