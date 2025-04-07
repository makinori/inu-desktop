package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
)

const WEB_PORT = 4845
const UDP_PORT = 4845

var (
	// localPort int
	localRtpPort int

	// localWebRTCAPI  *webrtc.API
	publicWebRTCAPI *webrtc.API

	streamVideoTrack *webrtc.TrackLocalStaticRTP
	streamAudioTrack *webrtc.TrackLocalStaticRTP

	peerConfig = webrtc.Configuration{
		// shouldnt need to stun when using nat1to1
		// ICEServers: []webrtc.ICEServer{
		// 	{
		// 		URLs: []string{"stun:stun.l.google.com:19302"},
		// 	},
		// },
	}

	// localServeMux  *http.ServeMux
	publicServeMux *http.ServeMux

	// sdpH264RegExp = regexp.MustCompile("(?i)rtpmap:([0-9]+) h264")
	// sdpOpusRegExp = regexp.MustCompile("(?i)rtpmap:([0-9]+) opus")
)

func mustSetupWebRTC() {
	// setup api

	mediaEngine := &webrtc.MediaEngine{}

	err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeH264,
			ClockRate:    90000,
			Channels:     0,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo)

	if err != nil {
		panic(err)
	}

	err = mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeOpus,
			ClockRate:    48000,
			Channels:     2,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio)

	if err != nil {
		panic(err)
	}

	// // user configurable RTP/RTCP Pipeline
	// // provides NACKs, RTCP Reports and other features
	// interceptorRegistry := &interceptor.Registry{}

	// // this sends a PLI every 3 seconds
	// // a PLI causes a video keyframe to be generated by the sender
	// // this makes our video seekable and more error resilent,
	// // but at a cost of lower picture quality and higher bitrates
	// intervalPliFactory, err := intervalpli.NewReceiverInterceptor()
	// if err != nil {
	// 	panic(err)
	// }

	// interceptorRegistry.Add(intervalPliFactory)

	// setup public setting engine

	publicUDPMux, err := ice.NewMultiUDPMuxFromPort(UDP_PORT)
	if err != nil {
		panic(err)
	}
	log.Infof("public udp listening at %d", UDP_PORT)

	publicSettingEngine := webrtc.SettingEngine{}
	publicSettingEngine.SetLite(true)
	publicSettingEngine.SetICEUDPMux(publicUDPMux)
	publicSettingEngine.SetIncludeLoopbackCandidate(false)
	publicSettingEngine.SetInterfaceFilter(func(s string) (keep bool) {
		return false
	})
	publicSettingEngine.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP4,
	})
	publicSettingEngine.SetNAT1To1IPs(
		[]string{
			"162.233.34.155", // TODO: use dns??
		},
		webrtc.ICECandidateTypeHost,
	)

	// setup local setting engine

	// localPort, err = getFreePort()
	// if err != nil {
	// 	panic(err)
	// }

	// localUDPMux, err := ice.NewMultiUDPMuxFromPort(
	// 	localPort, ice.UDPMuxFromPortWithLoopback(),
	// )
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("local udp listening at %d\n", localPort)

	// localSettingEngine := webrtc.SettingEngine{}
	// localSettingEngine.SetLite(true)
	// localSettingEngine.SetICEUDPMux(localUDPMux)
	// localSettingEngine.SetIncludeLoopbackCandidate(true)
	// localSettingEngine.SetInterfaceFilter(func(s string) (keep bool) {
	// 	return false
	// })
	// localSettingEngine.SetNetworkTypes([]webrtc.NetworkType{
	// 	webrtc.NetworkTypeUDP4,
	// })
	// // TODO: gstreamer cant connect to loopback candidate
	// // commenting below will let all interfaces candidate
	// localSettingEngine.SetNAT1To1IPs([]string{"127.0.0.1"},
	// 	webrtc.ICECandidateTypeHost,
	// )

	// make webrtc apis

	// localWebRTCAPI = webrtc.NewAPI(
	// 	webrtc.WithMediaEngine(mediaEngine),
	// 	webrtc.WithSettingEngine(localSettingEngine),
	// )

	publicWebRTCAPI = webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(publicSettingEngine),
		// webrtc.WithInterceptorRegistry(interceptorRegistry),
	)

	// setup tracks

	streamVideoTrack, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeH264,
	}, "video", "inu")

	if err != nil {
		panic(err)
	}

	streamAudioTrack, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeOpus,
	}, "audio", "inu")

	if err != nil {
		panic(err)
	}
}

func writeAnswer(
	w http.ResponseWriter, r *http.Request, peer *webrtc.PeerConnection,
	offer []byte, path string,
) {
	peer.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Info(getRequestIP(r) + " " + state.String())

		if state == webrtc.ICEConnectionStateFailed {
			peer.Close()
		}
	})

	err := peer.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  string(offer),
	})

	if err != nil {
		peer.Close()
		panic(err)
	}

	// TODO: should trickle
	gatherComplete := webrtc.GatheringCompletePromise(peer)

	answer, err := peer.CreateAnswer(nil)
	if err != nil {
		peer.Close()
		panic(err)
	}

	err = peer.SetLocalDescription(answer)
	if err != nil {
		peer.Close()
		panic(err)
	}

	<-gatherComplete

	w.Header().Add("Location", path)
	w.WriteHeader(http.StatusCreated)

	// uncomment to see server's sdp
	// fmt.Println(peer.LocalDescription().SDP)

	fmt.Fprint(w, peer.LocalDescription().SDP)
}

// func localWhipHandler(w http.ResponseWriter, r *http.Request) {
// 	offer, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println("got whip")

// 	offerStr := string(offer)

// 	var videoPayloadType uint8
// 	var audioPayloadType uint8

// 	h264Matches := sdpH264RegExp.FindStringSubmatch(offerStr)
// 	if len(h264Matches) > 0 {
// 		value, _ := strconv.Atoi(h264Matches[1])
// 		videoPayloadType = uint8(value)
// 	}

// 	opusMatches := sdpOpusRegExp.FindStringSubmatch(offerStr)
// 	if len(opusMatches) > 0 {
// 		value, _ := strconv.Atoi(opusMatches[1])
// 		audioPayloadType = uint8(value)
// 	}

// 	// offerStr = strings.ReplaceAll(offerStr, "192.168.1.36", "127.0.0.1")
// 	// fmt.Println(offerStr)
// 	// offer = []byte(offerStr)

// 	peer, err := localWebRTCAPI.NewPeerConnection(peerConfig)
// 	if err != nil {
// 		panic(err)
// 	}

// 	_, err = peer.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo)
// 	if err != nil {
// 		panic(err)
// 	}

// 	_, err = peer.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio)
// 	if err != nil {
// 		panic(err)
// 	}

// 	peer.OnTrack(func(track *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
// 		for {
// 			pkt, _, err := track.ReadRTP()
// 			if err != nil {
// 				return
// 			}

// 			switch pkt.PayloadType {
// 			case videoPayloadType:
// 				streamVideoTrack.WriteRTP(pkt)
// 			case audioPayloadType:
// 				streamAudioTrack.WriteRTP(pkt)
// 			}
// 		}
// 	})

// 	writeAnswer(w, peer, offer, "/whip")
// }

func publicWhepHandler(w http.ResponseWriter, r *http.Request) {
	offer, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	peer, err := publicWebRTCAPI.NewPeerConnection(peerConfig)
	if err != nil {
		panic(err)
	}

	rtpVideoSender, err := peer.AddTrack(streamVideoTrack)
	if err != nil {
		peer.Close()
		panic(err)
	}

	rtpAudioSender, err := peer.AddTrack(streamAudioTrack)
	if err != nil {
		peer.Close()
		panic(err)
	}

	// read incoming RTCP packets
	// before these packets are returned, they are processed by interceptors.
	// for things like NACK this needs to be called.
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if peer.ConnectionState() == webrtc.PeerConnectionStateClosed {
				break
			}
			rtpVideoSender.Read(rtcpBuf)
			rtpAudioSender.Read(rtcpBuf)
		}
	}()

	writeAnswer(w, r, peer, offer, "/whep")
}

func initWebRTC() {
	mustSetupWebRTC()

	publicServeMux = http.NewServeMux()
	publicServeMux.HandleFunc("POST /whep", publicWhepHandler)
	publicServeMux.Handle("/", http.FileServer(http.Dir(".")))

	log.Infof("public http listening at http://127.0.0.1:%d", WEB_PORT)
	go func() {
		err := http.ListenAndServe(":"+strconv.Itoa(WEB_PORT), publicServeMux)
		if err != nil {
			panic(err)
		}
	}()

	// localServeMux = http.NewServeMux()
	// localServeMux.HandleFunc("POST /whip", localWhipHandler)

	// fmt.Println("local http listening at http://127.0.0.1:" + strconv.Itoa(localPort))
	// go func() {
	// 	err := http.ListenAndServe(":"+strconv.Itoa(localPort), localServeMux)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }()

	var err error
	localRtpPort, err = getFreeUDPPort()
	if err != nil {
		panic(err)
	}

	log.Infof("local rtp listening at %d", localRtpPort)

	go func() {
		l, err := net.ListenUDP("udp", &net.UDPAddr{
			IP: net.ParseIP("127.0.0.1"), Port: localRtpPort,
		})
		if err != nil {
			panic(err)
		}

		bufferSize := 300000 // 300KB
		err = l.SetReadBuffer(bufferSize)
		if err != nil {
			panic(err)
		}

		defer func() {
			err = l.Close()
			if err != nil {
				panic(err)
			}
		}()

		packet := make([]byte, 1600)
		for {
			n, _, err := l.ReadFrom(packet)
			if err != nil {
				log.Error("error during read: %s\n", err)
				continue
			}

			streamVideoTrack.Write(packet[:n])

		}
	}()
}
