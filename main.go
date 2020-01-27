package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/pion/webrtc"
	"golang.org/x/net/websocket"
)

func checkPanic(err error) {
	if err != nil {
		panic(err)
	}
}

type l2peer struct{}

type minediveClient struct {
	ID      uint64
	Name    string
	SK      [32]byte
	PK      [32]byte
	Nonce   [24]byte
	ws      *websocket.Conn
	ticker  *time.Ticker
	done    chan bool
	minL1   int
	minL2   int
	l1Peers []l1peer
	l2Peers []l2peer
	origin  string
	url     string
}

func newMinediveClient(url string) *minediveClient {
	m := new(minediveClient)
	m.origin = "chrome-extension://ndhgbicdmoiobbemdoimagofjlmgcbna"
	m.url = url
	m.Name = randName(12)
	m.ticker = time.NewTicker(1 * time.Second)
	m.minL1 = 1
	m.done = make(chan bool)

	return m
}

func randName(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rand.Seed(time.Now().UnixNano())
	if n < 3 {
		return ""
	}
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func wsGetPeers(mc *minediveClient) {
	var msg getPeersMsg
	msg.Name = mc.Name
	msg.Type = "getpeers"
	msg.ID = mc.ID
	websocket.JSON.Send(mc.ws, msg)
}

func getPeers(mc *minediveClient) {
	if len(mc.l1Peers) < mc.minL1 {
		wsGetPeers(mc)
	}
	if len(mc.l2Peers) < mc.minL2 {
		//askL2()
	}
}

func repeat(mc *minediveClient) {
	for {
		select {
		case <-mc.done:
			log.Println("closing repeat")
			return
		case t := <-mc.ticker.C:
			//fmt.Println("Tick at", t)
			_ = t
			getPeers(mc)
		}
	}
}

func (m *minediveClient) getPeerByName(name string) (*l1peer, error) {
	for _, p := range m.l1Peers {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, errors.New("peer not found")
}

func (m *minediveClient) acceptOffer(target, sdp string) {
	p, err := m.getPeerByName(target)
	checkPanic(err)
	log.Println("ACCEPT OFFER SIG STATE:", p.pc.SignalingState())
	desc := webrtc.SessionDescription{}
	desc.SDP = sdp
	desc.Type = webrtc.SDPTypeOffer
	p.pc.SetRemoteDescription(desc)
	offer, err := p.pc.CreateAnswer(nil)
	checkPanic(err)
	msg := offerMsg{
		Type:   "answer",
		Name:   m.Name,
		Target: target,
		SDP:    offer.SDP,
	}
	p.pc.SetLocalDescription(offer)
	log.Println("SEND ANSWER", msg.SDP)
	log.Println("DIFFER FROM?", p.pc.LocalDescription().SDP)
	websocket.JSON.Send(m.ws, msg)
}

func (m *minediveClient) acceptAnswer(target, sdp string) {
	p, err := m.getPeerByName(target)
	checkPanic(err)
	if p.pc.SignalingState() == webrtc.SignalingStateHaveLocalOffer {
		log.Println(p.pc.SignalingState())
		return
	}
	log.Println("ACCEPT ANSWER SIG STATE:", p.pc.SignalingState())
	desc := webrtc.SessionDescription{}
	desc.SDP = sdp
	desc.Type = webrtc.SDPTypeOffer
	log.Println(p.dc.ID())
	p.pc.SetRemoteDescription(desc)
}

func (m *minediveClient) sendOffer(target string) {
	var err error
	p, err := m.getPeerByName(target)
	checkPanic(err)
	offer, err := p.pc.CreateOffer(nil)
	p.pc.SetLocalDescription(offer)
	checkPanic(err)
	log.Println("SEND OFFER", offer.SDP)
	log.Println("DIFFER FROM?", p.pc.LocalDescription().SDP)
	//wait for backchannel
	msg := offerMsg{
		Type:   "offer",
		Name:   m.Name,
		Target: target,
		SDP:    offer.SDP,
	}
	websocket.JSON.Send(m.ws, msg)
}

func (m *minediveClient) DialWebsocket() {
	var err error

	m.ws, err = websocket.Dial(m.url, "json", m.origin)
	checkPanic(err)
	go repeat(m)
}

func (m *minediveClient) dumpPeers() {
	log.Println("begin dump")
	for i, p := range m.l1Peers {
		log.Println("dump", i, p.Name)
	}
	log.Println("end dump")
}

func minediveClientLoop(mc *minediveClient, results chan int) {
	for {
		var jmsg []byte
		var imsg idMsg
		websocket.Message.Receive(mc.ws, &jmsg)
		if jmsg != nil {
			var err error
			err = json.Unmarshal(jmsg, &imsg)
			if err != nil {
				log.Println(err.Error())
			} else {
				//log.Println(imsg.Type)
				switch imsg.Type {
				case "id":
					mc.ID = imsg.ID
					var umsg usernameMsg
					umsg.Type = "username"
					umsg.ID = mc.ID
					umsg.Name = mc.Name
					umsg.PK = b64.StdEncoding.EncodeToString(mc.PK[:])
					websocket.JSON.Send(mc.ws, umsg)
					log.Println("Registering as", mc.Name, "with ID", mc.ID)
				case "username":
					log.Println("Need to change Username")
				case "userlist":
					var umsg userlistMsg
					err = json.Unmarshal(jmsg, &umsg)
					checkPanic(err)
					_, err = mc.getPeerByName(umsg.Users[0].Name)
					if err != nil {
						p := newL1Peer(umsg.Users[0].Name)
						mc.l1Peers = append(mc.l1Peers, *p)
						//mc.dumpPeers()
						if umsg.Contact == 1 {
							//log.Println("peer to contact:", umsg.Users[0].Name)
							_true := true
							opts := webrtc.DataChannelInit{
								Negotiated: &_true,
								Protocol:   strPtr("json"),
							}
							p.dc, err = p.pc.CreateDataChannel("json", &opts)
							checkPanic(err)
							mc.sendOffer(umsg.Users[0].Name)
						} else {
							//log.Println(umsg.Users[0].Name, "will contact me.")
						}
					}
				case "answer":
					var omsg offerMsg
					err = json.Unmarshal(jmsg, &omsg)
					checkPanic(err)
					log.Println("Answer from", omsg.Name)
					log.Println("ANSWER", omsg.SDP)
					mc.acceptAnswer(omsg.Name, omsg.SDP)
				case "offer":
					var omsg offerMsg
					err = json.Unmarshal(jmsg, &omsg)
					checkPanic(err)
					log.Println("Offer from", omsg.Name)
					_, err = mc.getPeerByName(omsg.Name)
					if err != nil {
						log.Println("adding peer", omsg.Name)
						p := newL1Peer(omsg.Name)
						mc.l1Peers = append(mc.l1Peers, *p)
						log.Println(omsg.Name, "added")
					}
					log.Println("ACCEPT OFFER", omsg.SDP)
					mc.acceptOffer(omsg.Name, omsg.SDP)
				case "close":
					mc.ws.Close()
					mc.done <- false
					results <- 0
				default:
				}
			}
		} else {
			mc.ws.Close()
		}
	}
}

func main() {
	workersNum := flag.Int("workers", 1, "number of workers")
	urlStr := flag.String("url", "ws://localhost:6501", "target Websocket URL")
	flag.Parse()

	results := make(chan int, *workersNum)
	for n := 0; n < *workersNum; n++ {
		mc := newMinediveClient(*urlStr)
		mc.DialWebsocket()
		go minediveClientLoop(mc, results)
	}
	for a := 0; a < *workersNum; a++ {
		<-results
	}
}
