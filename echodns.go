package main

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func strategyMaker(name string, qtype uint16) uint16 {
	subdomain := strings.ToLower(strings.Split(name, ".")[0])
	if qtype == dns.TypeA {
		if strings.Contains(subdomain, "fwd") {
			return 1 // return rdns ip in cname
		} else if strings.Contains(subdomain, "rdns") {
			return 2 // return honey cname record
		} else if strings.Contains(subdomain, "honey") {
			return 3 // return timestamp in a record
		} else if strings.Contains(subdomain, "echo") {
			return 4 // basic echodns
		} else if strings.Contains(subdomain, "ttl") {
			return 5 // ttl test
		}
	}
	return 0
}

func InttoIPv4(n uint32) net.IP {
	b0 := (n >> 24) & 0xff
	b1 := (n >> 16) & 0xff
	b2 := (n >> 8) & 0xff
	b3 := n & 0xff
	return net.IPv4(byte(b0), byte(b1), byte(b2), byte(b3))
}

func TtlParser(domain string) uint32 {
	subdomain := strings.ToLower(strings.Split(domain, ".")[0])
	ttl, _ := strconv.Atoi(strings.Split(subdomain, "-")[0])
	return uint32(ttl)
}

func handleReflect(w dns.ResponseWriter, r *dns.Msg) {
	var (
		ip    net.IP
		port  int
		id    uint16
		name  string
		qtype uint16
	)
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true
	m.Authoritative = true
	if addr, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		ip = addr.IP
		port = addr.Port
	}
	id = m.MsgHdr.Id
	name = m.Question[0].Name
	qtype = m.Question[0].Qtype
	log.Log().Str("sip", ip.String()).Int64("port", int64(port)).Int64("id", int64(id)).Str("name", name).Int64("qtype", int64(qtype)).Msg("")
	//log.Printf("%v|%v|%v|%v|%v", ip, port, id, name, qtype)
	//fmt.Println(ip)
	//fmt.Println(name)
	//fmt.Println(qtype)
	switch strategyMaker(name, qtype) {
	case 1:
		cname_subdomain := "rdns-" + strings.Replace(ip.String(), ".", "-", -1)
		cname_fqdn := cname_subdomain + ".echodns.xyz."
		cname := &dns.CNAME{
			Hdr:    dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 14400},
			Target: cname_fqdn,
		}
		//fmt.Println(name+" "+cname_fqdn)
		m.Answer = append(m.Answer, cname)
	case 2:
		cname_fqdn := "honey.echodns.xyz."
		cname := &dns.CNAME{
			Hdr:    dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 14400},
			Target: cname_fqdn,
		}
		//fmt.Println(cname_fqdn)
		m.Answer = append(m.Answer, cname)
	case 3:
		time_str := strconv.FormatInt(time.Now().UnixMicro(), 10)
		time_int, _ := strconv.Atoi(time_str[5 : len(time_str)-2])
		time_int += rand.Intn(10000)
		timestamp := InttoIPv4(uint32(time_int))
		a := &dns.A{
			Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 14400},
			A:   timestamp,
		}
		m.Answer = append(m.Answer, a)
	case 4:
		a := &dns.A{
			Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
			A:   ip,
		}
		m.Answer = append(m.Answer, a)
	case 5:
		query_ttl := TtlParser(name)
		//fmt.Println(query_ttl)
		a := &dns.A{
			Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: query_ttl},
			A:   ip,
		}
		m.Answer = append(m.Answer, a)
	case 0:
		return
	}
	w.WriteMsg(m)
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	dns.HandleFunc("echodns.xyz.", handleReflect)
	server := &dns.Server{Addr: ":53", Net: "udp"}
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Failed to set up dns server!")
		panic(err)
	}
}
