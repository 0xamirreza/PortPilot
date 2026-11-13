package gui

import (
	"image"
	"math"
	"net/netip"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"github.com/0xamirreza/portpilot/internal/app"
	"github.com/0xamirreza/portpilot/internal/core/ports"
	"github.com/0xamirreza/portpilot/internal/gui/theme"
)

func TestProjectEquirectangular(t *testing.T) {
	w, h := 360.0, 180.0
	offX, offY := 0.0, 0.0

	// (0, 0) should be at center: x = 180, y = 90
	pt := projectEquirectangular(0, 0, w, h, offX, offY)
	if math.Abs(float64(pt.X)-180.0) > 0.001 || math.Abs(float64(pt.Y)-90.0) > 0.001 {
		t.Errorf("projectEquirectangular(0,0) = (%v, %v), want (180, 90)", pt.X, pt.Y)
	}
}

func TestSimpleWorldMapDataset(t *testing.T) {
	if len(SimpleWorldMapCountries) < 180 {
		t.Fatalf("expected at least 180 countries in SimpleWorldMapCountries, got %d", len(SimpleWorldMapCountries))
	}

	keyCodes := []string{"IR", "FI", "DE", "US", "GB", "FR", "RU", "CN", "JP", "IN", "CA", "AU", "BR", "NL", "GLOBAL"}
	for _, code := range keyCodes {
		c := GetMapCountry(code)
		if c == nil {
			t.Errorf("GetMapCountry(%q) returned nil", code)
			continue
		}
		if c.Center[0] <= 0 || c.Center[0] >= 1 || c.Center[1] <= 0 || c.Center[1] >= 1 {
			t.Errorf("country %q center out of bounds [0, 1]: %v", code, c.Center)
		}
	}
}

func TestWorldGeolocationRender(t *testing.T) {
	th, err := theme.Load()
	if err != nil {
		t.Fatalf("theme.Load failed: %v", err)
	}
	ui := &UI{
		th:      th,
		topoTab: 1, // World Geolocation
		geoZoom: 1.0,
	}

	testSockets := []ports.Socket{
		{
			Protocol:      ports.TCP,
			LocalIP:       netip.MustParseAddr("192.168.1.5"),
			LocalPort:     40001,
			RemoteIP:      netip.MustParseAddr("95.216.1.1"), // Finland
			RemotePort:    443,
			Process:       "firefox",
			State:         "ESTABLISHED",
			BytesKnown:    true,
			BytesSent:     50000,
			BytesReceived: 120000,
		},
		{
			Protocol:      ports.TCP,
			LocalIP:       netip.MustParseAddr("192.168.1.5"),
			LocalPort:     40002,
			RemoteIP:      netip.MustParseAddr("148.251.1.1"), // Germany
			RemotePort:    443,
			Process:       "chrome",
			State:         "ESTABLISHED",
			BytesKnown:    true,
			BytesSent:     20000,
			BytesReceived: 80000,
		},
		{
			Protocol:      ports.TCP,
			LocalIP:       netip.MustParseAddr("192.168.1.5"),
			LocalPort:     40003,
			RemoteIP:      netip.MustParseAddr("5.112.1.1"), // Iran
			RemotePort:    8080,
			Process:       "node",
			State:         "ESTABLISHED",
			BytesKnown:    true,
			BytesSent:     15000,
			BytesReceived: 35000,
		},
		{
			Protocol:      ports.TCP,
			LocalIP:       netip.MustParseAddr("192.168.1.5"),
			LocalPort:     40004,
			RemoteIP:      netip.MustParseAddr("10.0.0.1"), // private/local LAN
			RemotePort:    80,
			Process:       "curl",
			State:         "ESTABLISHED",
		},
	}

	view := app.View{
		Sockets: testSockets,
	}

	// Test 1: Full unzoomed flat map (1.0x)
	{
		ui.geoZoom = 1.0
		ui.geoPanX = 0
		ui.geoPanY = 0
		var ops op.Ops
		gtx := layout.Context{
			Ops:         &ops,
			Constraints: layout.Exact(image.Pt(1024, 768)),
		}
		_ = ui.topology(gtx, view)
	}

	// Test 2: Zoomed in (2.5x) with panning
	{
		ui.geoZoom = 2.5
		ui.geoPanX = -120.0
		ui.geoPanY = 50.0
		var ops op.Ops
		gtx := layout.Context{
			Ops:         &ops,
			Constraints: layout.Exact(image.Pt(1024, 768)),
		}
		_ = ui.topology(gtx, view)
	}

	// Test 3: Deep zoom (5.0x) focusing a country
	{
		ui.geoZoom = 5.0
		ui.geoSelectedCountry = "FI"
		var ops op.Ops
		gtx := layout.Context{
			Ops:         &ops,
			Constraints: layout.Exact(image.Pt(1024, 768)),
		}
		_ = ui.topology(gtx, view)
	}

	// Test 4: Scope filters ("FOREIGN" vs "INTERNAL")
	{
		ui.geoScopeFilter = "FOREIGN"
		var ops op.Ops
		gtx := layout.Context{
			Ops:         &ops,
			Constraints: layout.Exact(image.Pt(1024, 768)),
		}
		_ = ui.topology(gtx, view)

		ui.geoScopeFilter = "INTERNAL"
		ops.Reset()
		_ = ui.topology(gtx, view)
	}

	// Test 5: Empty sockets (0 flows)
	{
		emptyView := app.View{}
		ui.geoZoom = 1.0
		ui.geoScopeFilter = ""
		ui.geoSelectedCountry = ""
		var ops op.Ops
		gtx := layout.Context{
			Ops:         &ops,
			Constraints: layout.Exact(image.Pt(1024, 768)),
		}
		_ = ui.topology(gtx, emptyView)
	}

	// Test 6: Body and Sidebar layout rendering with active map zoom (verifying sidebar is unblocked)
	{
		ui.screen = 8 // screen Topology
		ui.topoTab = 1 // World Geolocation
		ui.geoZoom = 3.0
		ui.geoPanX = 100
		ui.geoPanY = -50
		var ops op.Ops
		gtx := layout.Context{
			Ops:         &ops,
			Constraints: layout.Exact(image.Pt(1024, 768)),
		}
		_ = ui.rail(gtx, view)
		_ = ui.body(gtx, view)
	}
}
