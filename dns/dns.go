package dns

import (
	"fmt"
	"log"
    "net"
    "encoding/binary"
    "regexp"
    "strconv"
    "io/ioutil"

	"github.com/miekg/dns"
    "github.com/bogdanovich/dns_resolver"
)

var resolver *dns_resolver.DnsResolver

func ip2int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

/*
 *  Ganerate IP
 */
func GenerateIP(instance uint32) net.IP {
    IP := net.ParseIP("127.0.0.0")
    IPint := ip2int(IP)
    IPint += instance
    IP = int2ip(IPint)
    
    return IP
}

/*
 *  Parse localhost
 */
func parseLocalhost(str string) (bool) {
    var expr = regexp.MustCompile(`^localhost\.$`)
	parts := expr.FindStringSubmatch(str)

    if len(parts) == 0 {
        return false
	}
        
    return true
}


/*
 *  Parse %d.localhost.
 */
func parseLocalhostInstance(str string) (uint32) {
    var expr = regexp.MustCompile(`^([0-9]+)\.localhost\.$`)
	parts := expr.FindStringSubmatch(str)

    if len(parts) == 0 {
        return 0
	}
        
    instance, err := strconv.Atoi(parts[1])
    if err != nil {
        return 0
    }
    
    // Range for localhost IP address is 2^24 because localhost subnet is 127.0.0.1/8
    // 2^24 = 16777216
    if instance >= 16777216 {
        return 0
    }
    
    return uint32(instance)
}

func parseQuery(m *dns.Msg) {
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
            ok := parseLocalhost(q.Name)
            if ok {
                rr, err := dns.NewRR(fmt.Sprintf("%s A 127.0.0.1", q.Name))
                if err == nil {
                    m.Answer = append(m.Answer, rr)
                }
                return
            }
            
            instance := parseLocalhostInstance(q.Name)
            if instance > 0 {
                IP := GenerateIP(instance)
                rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, IP.String()))
                if err == nil {
                    m.Answer = append(m.Answer, rr)
                }     
                return
            }

            // Check if host exists
            ipArr, err := resolver.LookupHost(q.Name)
            if err != nil {
                log.Println(err)
            }                

            for _, IP := range ipArr {
                rr, _ := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, IP.String()))
                m.Answer = append(m.Answer, rr)
            } 
		}
	}
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m)
	}

	w.WriteMsg(m)
}

func Start() {
	// attach request handler func
	dns.HandleFunc(".", handleDnsRequest)

    var err error
    
    resolver, err = dns_resolver.NewFromResolvConf("/etc/resolv.conf")
    if err != nil {
        log.Println(err)
    }
    
    nameserver := []byte("nameserver 127.0.0.1\n")
    err = ioutil.WriteFile("/etc/resolv.conf", nameserver, 0644)
	if err != nil {
		log.Printf("Failed to start server: %s\n ", err.Error())
	}
    
	// start server
    server := &dns.Server{Addr: net.JoinHostPort("127.0.0.1", "53"), Net: "udp"}
	log.Println("Starting at ", server.Addr)
    go func() {
        err = server.ListenAndServe()
        if err != nil {
            log.Printf("Failed to start server: %s\n ", err.Error())
        }        
    }()
}
