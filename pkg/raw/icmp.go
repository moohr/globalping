package raw

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type ICMP4TransceiverConfig struct {
	// ICMP ID to use
	ID int

	// Where to ping to
	Dst net.IPAddr

	PktTimeout time.Duration
}

type ICMPSendRequest struct {
	Seq int
	TTL int
}

type ICMPReceiveReply struct {
	Seq int
	TTL int

	// it's basically the same as Addr.Network() + ':' + Addr.String()
	Peer string

	ReceivedAt time.Time

	ICMPTypeV4 ipv4.ICMPType
}

type ICMP4Transceiver struct {
	id             int
	packetConn     net.PacketConn
	ipv4PacketConn *ipv4.PacketConn
	pktTimeout     time.Duration
	Dst            net.IPAddr

	// User send requests to here, we retrieve the request,
	// then we translate it to the wire format.
	SendC chan ICMPSendRequest

	// ICMP replies
	ReceiveC chan ICMPReceiveReply
}

func NewICMP4Transceiver(config ICMP4TransceiverConfig) (*ICMP4Transceiver, error) {

	if config.Dst.IP.To4() == nil {
		return nil, fmt.Errorf("destination is not an IPv4 address")
	}

	tracer := &ICMP4Transceiver{
		SendC:      make(chan ICMPSendRequest),
		ReceiveC:   make(chan ICMPReceiveReply),
		pktTimeout: config.PktTimeout,
		Dst:        config.Dst,
	}

	return tracer, nil
}

func getMaximumMTU() int {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	maximumMTU := -1
	for _, iface := range ifaces {
		if iface.MTU > maximumMTU {
			maximumMTU = iface.MTU
		}
	}
	if maximumMTU == -1 {
		panic("can't determine maximum MTU")
	}
	return maximumMTU
}

func (icmp4tr *ICMP4Transceiver) Run(ctx context.Context) error {
	conn, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return fmt.Errorf("failed to listen on packet:icmp: %v", err)
	}
	go func() {
		defer conn.Close()

		icmp4tr.packetConn = conn
		icmp4tr.ipv4PacketConn = ipv4.NewPacketConn(conn)

		if err := icmp4tr.ipv4PacketConn.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
			log.Fatal(err)
		}

		wm := icmp.Message{
			Type: ipv4.ICMPTypeEcho, Code: 0,
			Body: &icmp.Echo{
				ID:   icmp4tr.id,
				Data: nil,
			},
		}

		bufSize := getMaximumMTU()
		rb := make([]byte, bufSize)

		// launch receiving goroutine
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					err := icmp4tr.ipv4PacketConn.SetReadDeadline(time.Now().Add(icmp4tr.pktTimeout))
					if err != nil {
						log.Fatalf("failed to set read deadline: %v", err)
					}
					nBytes, ctrlMsg, peerAddr, err := icmp4tr.ipv4PacketConn.ReadFrom(rb)
					if err != nil {
						if err, ok := err.(net.Error); ok && err.Timeout() {
							log.Printf("timeout reading from connection, skipping")
							continue
						}
						log.Fatalf("failed to read from connection: %v", err)
					}
					receiveMsg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), rb[:nBytes])
					if err != nil {
						log.Fatalf("failed to parse icmp message: %v", err)
					}

					receivedAt := time.Now()
					replyObject := ICMPReceiveReply{
						ReceivedAt: receivedAt,
						Peer:       peerAddr.Network() + ":" + peerAddr.String(),
						TTL:        ctrlMsg.TTL,
						Seq:        -1, // if can't determine, use -1
					}

					icmpBody, ok := receiveMsg.Body.(*icmp.Echo)
					if !ok {
						log.Printf("failed to parse icmp body: %+v", receiveMsg)
						continue
					}

					if icmpBody.ID != icmp4tr.id {
						// silently ignore the message that is not for us
						continue
					}

					replyObject.Seq = icmpBody.Seq

					switch receiveMsg.Type {
					case ipv4.ICMPTypeTimeExceeded:
						replyObject.ICMPTypeV4 = ipv4.ICMPTypeTimeExceeded
					case ipv4.ICMPTypeEchoReply:
						replyObject.ICMPTypeV4 = ipv4.ICMPTypeEchoReply
					default:
						log.Printf("unknown ICMP message: %+v", receiveMsg)
						continue
					}

					icmp4tr.ReceiveC <- replyObject
				}
			}
		}()

		// launch sending goroutine
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case req := <-icmp4tr.SendC:
					wm.Body.(*icmp.Echo).Seq = req.Seq
					wb, err := wm.Marshal(nil)
					if err != nil {
						log.Fatalf("failed to marshal icmp message: %v", err)
					}

					if err := icmp4tr.ipv4PacketConn.SetTTL(req.TTL); err != nil {
						log.Fatalf("failed to set TTL: %v", err)
					}

					dst := icmp4tr.Dst
					if _, err := icmp4tr.ipv4PacketConn.WriteTo(wb, nil, &dst); err != nil {
						log.Fatalf("failed to write to connection: %v", err)
					}
				}
			}
		}()

		<-ctx.Done()
	}()

	return nil
}
