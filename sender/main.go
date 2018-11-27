package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/pions/webrtc"
	"github.com/pions/webrtc/examples/util"
	"github.com/pions/webrtc/pkg/datachannel"
	"github.com/pions/webrtc/pkg/ice"
)

var received string

func main() {
	recievedBytes := make(chan []byte, 1024)
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

	// Create a datachannel with label 'data'
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	util.Check(err)

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState ice.ConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Register channel opening handling
	dataChannel.OnOpen(func() {
		fmt.Printf("Data channel '%s'-'%d' open\n", dataChannel.Label, dataChannel.ID)
		time.Sleep(1 * time.Second)
		// fmt.Printf("What do you want to send?\n")
		// err := dataChannel.Send(datachannel.PayloadString{Data: []byte(util.MustReadStdin())})
		// util.Check(err)
		fmt.Println("sending file")
		const BufferSize = 1024
		file, err := os.Open("sender.exe")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		buffer := make([]byte, BufferSize)

		var piece uint64
		piece = 100
		for {
			bytesread, err := file.Read(buffer)

			if err != nil {
				if err != io.EOF {
					fmt.Println(err)
				}

				break
			}

			pieceByte := make([]byte, 8)
			binary.LittleEndian.PutUint64(pieceByte, piece)
			dataToSend := append(pieceByte, buffer[:bytesread]...)
			for {
				err = dataChannel.Send(datachannel.PayloadBinary{Data: dataToSend})
				if err != nil {
					log.Println(err)
				}
				time.Sleep(10 * time.Millisecond)
				log.Printf("waiting for ack\n")
				doneWaiting := false
				select {
				case gotBytes := <-recievedBytes:
					doneWaiting = bytes.Equal(pieceByte, gotBytes)
				default:
					time.Sleep(10 * time.Millisecond)
				}
				if doneWaiting {
					break
				}
			}

			// for i := 0; i < 1000; i++ {
			// 	err := dataChannel.Send(datachannel.PayloadString{Data: []byte(fmt.Sprintf("%d", i))})
			// 	if err != nil {
			// 		log.Println(err)
			// 	}
			// 	time.Sleep(1 * time.Microsecond)
			// }

			piece += 1
		}
		log.Println("sending done signal")
		err = dataChannel.Send(datachannel.PayloadString{Data: []byte("done")})
		if err != nil {
			log.Println(err)
		}

		time.Sleep(3 * time.Second)
	})

	// Register the OnMessage to handle incoming messages
	dataChannel.OnMessage(func(payload datachannel.Payload) {
		switch p := payload.(type) {
		case *datachannel.PayloadString:
			fmt.Printf("Message '%s' from DataChannel '%s' payload '%s'\n", p.PayloadType().String(), dataChannel.Label, string(p.Data))
		case *datachannel.PayloadBinary:
			fmt.Printf("Message '%s' from DataChannel '%s' payload '% 02x'\n", p.PayloadType().String(), dataChannel.Label, p.Data)
			recievedBytes <- p.Data
		default:
			fmt.Printf("Message '%s' from DataChannel '%s' no payload \n", p.PayloadType().String(), dataChannel.Label)
		}
	})

	// Create an offer to send to the browser
	offer, err := peerConnection.CreateOffer(nil)
	util.Check(err)

	// Output the offer in base64 so we can paste it in browser
	fmt.Println(util.Encode(offer.Sdp))

	// Wait for the answer to be pasted
	sd := util.Decode(util.MustReadStdin())

	// Set the remote SessionDescription
	answer := webrtc.RTCSessionDescription{
		Type: webrtc.RTCSdpTypeAnswer,
		Sdp:  sd,
	}

	// Apply the answer as the remote description
	err = peerConnection.SetRemoteDescription(answer)
	util.Check(err)

	// Block forever
	select {}
}
