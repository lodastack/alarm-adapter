// Copyright 2009 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// taken from http://golang.org/src/pkg/net/ipraw_test.go

package ping

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/lodastack/log"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

//count of pings to send to each target
const PingTimes = 10

//interval between ping packets to one target (in millisec)
const IntervalPackage = 50

const (
	icmpv4EchoRequest = 8
	icmpv4EchoReply   = 0
	icmpv6EchoRequest = 128
	icmpv6EchoReply   = 129
	protocolIPv6ICMP  = 58
)

type icmpMessage struct {
	Type     int             // type
	Code     int             // code
	Checksum int             // checksum
	Body     icmpMessageBody // body
}

type icmpMessageBody interface {
	Len() int
	Marshal() ([]byte, error)
}

// Marshal returns the binary enconding of the ICMP echo request or
// reply message m.
func (m *icmpMessage) Marshal() ([]byte, error) {
	b := []byte{byte(m.Type), byte(m.Code), 0, 0}
	if m.Body != nil && m.Body.Len() != 0 {
		mb, err := m.Body.Marshal()
		if err != nil {
			return nil, err
		}
		b = append(b, mb...)
	}
	switch m.Type {
	case icmpv6EchoRequest, icmpv6EchoReply:
		return b, nil
	}
	csumcv := len(b) - 1 // checksum coverage
	s := uint32(0)
	for i := 0; i < csumcv; i += 2 {
		s += uint32(b[i+1])<<8 | uint32(b[i])
	}
	if csumcv&1 == 0 {
		s += uint32(b[csumcv])
	}
	s = s>>16 + s&0xffff
	s = s + s>>16
	// Place checksum back in header; using ^= avoids the
	// assumption the checksum bytes are zero.
	b[2] ^= byte(^s & 0xff)
	b[3] ^= byte(^s >> 8)
	return b, nil
}

// parseICMPMessage parses b as an ICMP message.
func parseICMPMessage(b []byte) (*icmpMessage, error) {
	msglen := len(b)
	if msglen < 4 {
		return nil, errors.New("message too short")
	}
	m := &icmpMessage{Type: int(b[0]), Code: int(b[1]), Checksum: int(b[2])<<8 | int(b[3])}
	if len(b) > 4 {
		var err error
		switch m.Type {
		case icmpv4EchoRequest, icmpv4EchoReply, icmpv6EchoRequest, icmpv6EchoReply:
			m.Body, err = parseICMPEcho(b[4:])
			if err != nil {
				return nil, err
			}
		}
	}
	return m, nil
}

// imcpEcho represenets an ICMP echo request or reply message body.
type icmpEcho struct {
	ID   int    // identifier
	Seq  int    // sequence number
	Data []byte // data
}

func (p *icmpEcho) Len() int {
	if p == nil {
		return 0
	}
	return 4 + len(p.Data)
}

// Marshal returns the binary enconding of the ICMP echo request or
// reply message body p.
func (p *icmpEcho) Marshal() ([]byte, error) {
	b := make([]byte, 4+len(p.Data))
	b[0], b[1] = byte(p.ID>>8), byte(p.ID&0xff)
	b[2], b[3] = byte(p.Seq>>8), byte(p.Seq&0xff)
	copy(b[4:], p.Data)
	return b, nil
}

// parseICMPEcho parses b as an ICMP echo request or reply message body.
func parseICMPEcho(b []byte) (*icmpEcho, error) {
	bodylen := len(b)
	p := &icmpEcho{ID: int(b[0])<<8 | int(b[1]), Seq: int(b[2])<<8 | int(b[3])}
	if bodylen > 4 {
		p.Data = make([]byte, bodylen-4)
		copy(p.Data, b[4:])
	}
	return p, nil
}

// Ping pings a IP adderss
// Para timeout unit is second.
func Ping(address string, timeout int) float64 {
	ipAddr, err := net.ResolveIPAddr("ip", address)
	if err != nil {
		return 0
	}
	var FailedCount float64
	for i := 1; i <= PingTimes; i++ {
		if isIPv4(ipAddr.IP) {
			err := v4Pinger(address, timeout)
			if err != nil {
				log.Error("ping response with msg:", err)
				FailedCount++
			}
		} else {
			err := v6Pinger(address, timeout)
			if err != nil {
				log.Error("ping response with msg:", err)
				FailedCount++
			}
		}
		time.Sleep(time.Duration(IntervalPackage) * time.Millisecond)
	}
	fmt.Println(FailedCount)
	return (FailedCount / PingTimes) * 100
}

func v4Pinger(address string, timeout int) error {
	c, err := net.DialTimeout("ip4:icmp", address, time.Duration(timeout)*time.Second)
	if err != nil {
		return err
	}
	if err = c.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Second)); err != nil {
		return err
	}
	defer c.Close()

	typ := icmpv4EchoRequest
	xid, xseq := os.Getpid()&0xffff, 1
	wb, err := (&icmpMessage{
		Type: typ, Code: 0,
		Body: &icmpEcho{
			ID: xid, Seq: xseq,
			Data: bytes.Repeat([]byte("Go Go Ping!!!"), 3),
		},
	}).Marshal()
	if err != nil {
		return err
	}
	if _, err = c.Write(wb); err != nil {
		return err
	}
	var m *icmpMessage

	for {
		rb := make([]byte, 20+len(wb))
		if _, err = c.Read(rb); err != nil {
			return err
		}
		rb = ipv4Payload(rb)
		if m, err = parseICMPMessage(rb); err != nil {
			return err
		}
		switch m.Type {
		case icmpv4EchoReply:
			return nil
		}
	}
	return nil
}

func ipv4Payload(b []byte) []byte {
	if len(b) < 20 {
		return b
	}
	hdrlen := int(b[0]&0x0f) << 2
	return b[hdrlen:]
}

func v6Pinger(address string, timeout int) error {
	c, err := net.DialTimeout("ip6:ipv6-icmp", address, time.Duration(timeout)*time.Second)
	if err != nil {
		return err
	}
	if err = c.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Second)); err != nil {
		return err
	}
	defer c.Close()

	typ := icmpv6EchoRequest
	xid, xseq := os.Getpid()&0xffff, 1
	wb, err := (&icmpMessage{
		Type: typ, Code: 0,
		Body: &icmpEcho{
			ID: xid, Seq: xseq,
			Data: bytes.Repeat([]byte("Go Go Ping!!!"), 3),
		},
	}).Marshal()
	if err != nil {
		return err
	}
	if _, err = c.Write(wb); err != nil {
		return err
	}

	for {
		rb := make([]byte, 20+len(wb))
		if _, err = c.Read(rb); err != nil {
			return err
		}
		var m *icmp.Message
		var err error
		if m, err = icmp.ParseMessage(protocolIPv6ICMP, rb); err != nil {
			return fmt.Errorf("Error parsing icmp v6 message")
		}
		switch m.Type {
		case ipv6.ICMPTypeEchoReply:
			return nil
		}
	}
	return nil
}

func isIPv4(ip net.IP) bool {
	v := ip.To4()
	if v != nil {
		return true
	}
	return false
}
