package containers

import "testing"

func TestParseDocker(t *testing.T) {
	text := `{"ID":"dda5980217ac99","Names":"kibana","Status":"Up","Image":"kibana","Ports":"127.0.0.1:9200->9200/tcp"}` + "\n"
	items := ParseDocker(text)
	if len(items) != 1 || items[0].ID != "dda5980217ac" || len(items[0].Published) != 1 || items[0].Published[0].Container != "9200" {
		t.Fatalf("%+v", items)
	}
}

func TestParsePublish(t *testing.T) {
	got := ParsePublish("127.0.0.1:9200->9200/tcp, 0.0.0.0:80->80/udp")
	if len(got) != 2 || got[0].Host != "127.0.0.1:9200" || got[1].Proto != "udp" {
		t.Fatalf("%+v", got)
	}
}

func TestAttachHostSocket(t *testing.T) {
	items := Attach([]Item{{Kind: "docker", ID: "dda5980217ac", Name: "kibana", Ports: "127.0.0.1:9200->9200/tcp"}}, []HostSocket{{
		Container: "docker-dda5980217ac", Process: "java", Proto: "tcp", Addr: "127.0.0.1:9200", Port: 9200,
	}})
	if items[0].Host == "" {
		t.Fatal(items[0])
	}
}

func TestParseVirshNet(t *testing.T) {
	text := " Name      State    Autostart   Persistent\n--------------------------------------------\n default   active   yes         yes\n"
	items := ParseVirsh("libvirt-net", text)
	if len(items) != 1 || items[0].Name != "default" {
		t.Fatalf("%+v", items)
	}
}
