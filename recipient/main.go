package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/pions/webrtc"
	"github.com/pions/webrtc/examples/util"
	"github.com/pions/webrtc/pkg/datachannel"
	"github.com/pions/webrtc/pkg/ice"
)

func main() {
	fmt.Println("ready")
	// Everything below is the pion-WebRTC API! Thanks for using it ❤️.

	// Prepare the configuration
	config := webrtc.RTCConfiguration{
		IceServers: []webrtc.RTCIceServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.New(config)
	util.Check(err)

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState ice.ConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Register data channel creation handling
	peerConnection.OnDataChannel(func(d *webrtc.RTCDataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label, d.ID)
		d.Send(datachannel.PayloadBinary{Data: []byte("ready")]})
		sendBytes := make(chan []byte, 1024)
		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label, d.ID)
			for {
				data := <-sendBytes
				err := d.Send(datachannel.PayloadBinary{Data: data})
				if err != nil {
					log.Println(err)
				}
				log.Println("sent")

			}
			// for {
			// 	fmt.Printf("What do you want to send?\n")
			// 	err := d.Send(datachannel.PayloadString{Data: []byte(util.MustReadStdin())})
			// 	util.Check(err)
			// }
		})

		f, err := os.Create("d.exe")
		if err != nil {
			panic(err)
		}

		// Register message handling
		d.OnMessage(func(payload datachannel.Payload) {
			switch p := payload.(type) {
			case *datachannel.PayloadString:
				fmt.Printf("Message '%s' from DataChannel '%s' payload '%s'\n", p.PayloadType().String(), d.Label, string(p.Data))
				if bytes.Equal(p.Data, []byte("done")) {
					f.Close()
					os.Exit(1)
				}
			case *datachannel.PayloadBinary:
				dataRecieved := p.Data
				fmt.Printf("received %d bytes\n", len(dataRecieved))
				f.Write(dataRecieved[8:])
				log.Println("sending ack")
				sendBytes <- dataRecieved[:8]
			default:
				fmt.Printf("Message '%s' from DataChannel '%s' no payload \n", p.PayloadType().String(), d.Label)
			}
		})
	})

	// Wait for the offer to be pasted
	offer := util.Decode(util.MustReadStdin())

	// Set the remote SessionDescription
	err = peerConnection.SetRemoteDescription(offer)
	util.Check(err)

	// Sets the LocalDescription, and starts our UDP listeners
	answer, err := peerConnection.CreateAnswer(nil)
	util.Check(err)

	// Output the answer in base64 so we can paste it in browser
	fmt.Println(util.Encode(answer))

	// Block forever
	select {}
}
