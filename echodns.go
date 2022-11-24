package main

import (
	"bytes"
	"encoding/binary"
	_ "fmt"
	"net"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func serveDNS(u *net.UDPConn, clientaddr net.Addr, request *layers.DNS) {
	replyMess := request
	var dnsAnswer layers.DNSResourceRecord
	if request == nil || request.Questions == nil || len(request.Questions) == 0 {
		return
	} else {
		dnsAnswer.Name = []byte(request.Questions[0].Name)
	}
	//fmt.Println(clientaddr.String())
	replyMess.QR = true
	replyMess.ANCount = 1
	replyMess.OpCode = layers.DNSOpCodeQuery
	replyMess.AA = true
	if request.Questions[0].Type == layers.DNSTypeA {
		dnsAnswer.Type = layers.DNSTypeA
		dnsAnswer.Class = layers.DNSClassIN
		dnsAnswer.IP = net.ParseIP(strings.Split(clientaddr.String(), ":")[0])
		dnsAnswer.TTL = 1000
	} else if request.Questions[0].Type == layers.DNSTypeAAAA {
		dnsAnswer.Type = layers.DNSTypeAAAA
		dnsAnswer.Class = layers.DNSClassIN
		t := time.Now().UnixMicro()
		rdns_ipv4 := net.ParseIP(strings.Split(clientaddr.String(), ":")[0]).To4()
		bytebuf := bytes.NewBuffer([]byte{})
		binary.Write(bytebuf, binary.BigEndian, t)
		binary.Write(bytebuf, binary.BigEndian, rdns_ipv4)
		binary.Write(bytebuf, binary.BigEndian, []byte{0, 0, 0, 0})
		dnsAnswer.IP = net.IP(bytebuf.Bytes())
		dnsAnswer.TTL = 1000
	}
	replyMess.Answers = append(replyMess.Answers, dnsAnswer)
	replyMess.ResponseCode = layers.DNSResponseCodeNoErr
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := replyMess.SerializeTo(buf, opts)
	if err != nil {
		panic(err)
	}
	u.WriteTo(buf.Bytes(), clientaddr)
}

func main() {
	addr := net.UDPAddr{
		Port: 53,
		IP:   net.ParseIP("localhost.localdomain"), //localhost
	}
	u, _ := net.ListenUDP("udp", &addr)

	for {
		tmp := make([]byte, 1024)
		_, addr, _ := u.ReadFrom(tmp)
		clientaddr := addr
		packet := gopacket.NewPacket(tmp, layers.LayerTypeDNS, gopacket.Default)
		dnsPacket := packet.Layer(layers.LayerTypeDNS)
		tcp, _ := dnsPacket.(*layers.DNS)
		serveDNS(u, clientaddr, tcp)
	}
}
