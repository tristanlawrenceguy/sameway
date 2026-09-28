package discover

import (
	"strings"
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

// A question is a plain mDNS query; an instance name is read back into a
// service with its type and its name, spaces and all.
func TestQuestionsAndNames(t *testing.T) {
	q, err := question("_services._dns-sd._udp.local.", dnsmessage.TypePTR)
	if err != nil || len(q) == 0 {
		t.Fatalf("a query is built: %v", err)
	}
	var p dnsmessage.Parser
	if _, err := p.Start(q); err != nil {
		t.Fatal(err)
	}
	qs, _ := p.AllQuestions()
	if len(qs) != 1 || qs[0].Name.String() != "_services._dns-sd._udp.local." || qs[0].Type != dnsmessage.TypePTR {
		t.Errorf("one PTR question for the services list, got %+v", qs)
	}

	found := map[string]*Service{}
	svc := entry(found, `Kitchen\032Light._hap._tcp.local.`, "")
	if svc.Type != "_hap._tcp" || svc.Name != "Kitchen Light" {
		t.Errorf("type and name from the instance, got %+v", svc)
	}
	svc = entry(found, "broker._mqtt._tcp.local", "_mqtt._tcp.local")
	svc.Host, svc.Port, svc.Addr = "pi.local", 1883, "192.168.1.10"
	if line := Describe(*svc); !strings.HasPrefix(line, "MQTT broker") || !strings.Contains(line, "192.168.1.10:1883") {
		t.Errorf("a broker is described as one, where it is: %q", line)
	}
}
