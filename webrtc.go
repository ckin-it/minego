package main

import (
	"fmt"
	"log"
	"time"

	"github.com/pion/webrtc"
)

func wrtc() {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	checkPanic(err)
	offer, err := peerConnection.CreateOffer(nil)
	checkPanic(err)
	log.Println(offer)

	peerConnection.OnICECandidate(func(ice *webrtc.ICECandidate) {
		desc := peerConnection.LocalDescription()
		peerConnection.SetLocalDescription(*desc)
	})

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Register data channel creation handling
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label(), d.ID())

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

	// Wait for the offer to be pasted
	//offer := webrtc.SessionDescription{}
	//log.Println(offer)

	//signalDecode(signalMustReadStdin(), &offer)

	// Set the remote SessionDescription
	//err = peerConnection.SetRemoteDescription(offer)
	//if err != nil {
	//	panic(err)
	//}

	// Create an answer
	//answer, err := peerConnection.CreateAnswer(nil)
	//if err != nil {
	//	panic(err)
	//}

	// Sets the LocalDescription, and starts our UDP listeners
	//err = peerConnection.SetLocalDescription(answer)
	//if err != nil {
	//	panic(err)
	//}

	// Output the answer in base64 so we can paste it in browser
	//fmt.Println(signalEncode(answer))

	// Block forever
	//select {}
}
