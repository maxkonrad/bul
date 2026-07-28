package scanner

import (
	"testing"
	"time"
)

func TestParseIPRange_SingleIP(t *testing.T) {
	ips, err := ParseIPRange("192.168.1.1")
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if len(ips) != 1 {
		t.Fatalf("1 IP bekleniyordu, %d alındı", len(ips))
	}
	if ips[0] != "192.168.1.1" {
		t.Errorf("beklenen: 192.168.1.1, alınan: %s", ips[0])
	}
}

func TestParseIPRange_Range(t *testing.T) {
	ips, err := ParseIPRange("192.168.1.1-5")
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if len(ips) != 5 {
		t.Fatalf("5 IP bekleniyordu, %d alındı", len(ips))
	}

	expected := []string{
		"192.168.1.1",
		"192.168.1.2",
		"192.168.1.3",
		"192.168.1.4",
		"192.168.1.5",
	}
	for i, ip := range ips {
		if ip != expected[i] {
			t.Errorf("indeks %d: beklenen %s, alınan %s", i, expected[i], ip)
		}
	}
}

func TestParseIPRange_FullRange(t *testing.T) {
	ips, err := ParseIPRange("10.0.0.0-255")
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if len(ips) != 256 {
		t.Fatalf("256 IP bekleniyordu, %d alındı", len(ips))
	}
	if ips[0] != "10.0.0.0" {
		t.Errorf("ilk IP: beklenen 10.0.0.0, alınan %s", ips[0])
	}
	if ips[255] != "10.0.0.255" {
		t.Errorf("son IP: beklenen 10.0.0.255, alınan %s", ips[255])
	}
}

func TestParseIPRange_InvalidFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
	}{
		{"eksik_oktet", "192.168.1"},
		{"fazla_oktet", "192.168.1.1.1"},
		{"gecersiz_deger", "192.168.1.abc"},
		{"oktet_disi_deger", "192.168.1.256"},
		{"negatif_oktet", "192.168.1.-1"},
		{"gecersiz_bitis", "192.168.1.1-abc"},
		{"bitis_disi_deger", "192.168.1.1-256"},
		{"ters_aralik", "192.168.1.10-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseIPRange(tt.input)
			if err == nil {
				t.Errorf("%q için hata bekleniyordu, hata alınmadı", tt.input)
			}
		})
	}
}

func TestExtractRTT(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{
			"normal",
			"64 bytes from 192.168.1.1: icmp_seq=1 ttl=64 time=1.23 ms",
			1230 * time.Microsecond,
		},
		{
			"kucuk_deger",
			"64 bytes from 192.168.1.1: icmp_seq=1 ttl=64 time=0.456 ms",
			456 * time.Microsecond,
		},
		{
			"tam_sayi",
			"64 bytes from 192.168.1.1: icmp_seq=1 ttl=64 time=5 ms",
			5 * time.Millisecond,
		},
		{
			"time_lesser",
			"64 bytes from 192.168.1.1: icmp_seq=1 ttl=64 time<1 ms",
			1 * time.Millisecond,
		},
		{
			"bulunamadi",
			"Request timeout for icmp_seq 0",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRTT(tt.input)
			if got != tt.expected {
				t.Errorf("beklenen: %v, alınan: %v", tt.expected, got)
			}
		})
	}
}

func TestParseIPNeighOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"reachable",
			"192.168.1.1 dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE",
			"AA:BB:CC:DD:EE:FF",
		},
		{
			"stale",
			"192.168.1.1 dev wlan0 lladdr 11:22:33:44:55:66 STALE",
			"11:22:33:44:55:66",
		},
		{
			"bos",
			"192.168.1.1 dev eth0 FAILED",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIPNeighOutput(tt.input)
			if got != tt.expected {
				t.Errorf("beklenen: %q, alınan: %q", tt.expected, got)
			}
		})
	}
}

func TestParseARPOutput(t *testing.T) {
	input := `Address         HWtype  HWaddress           Flags Mask  Iface
192.168.1.1     ether   aa:bb:cc:dd:ee:ff   C           eth0`

	got := parseARPOutput(input)
	expected := "AA:BB:CC:DD:EE:FF"
	if got != expected {
		t.Errorf("beklenen: %q, alınan: %q", expected, got)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Timeout != 2*time.Second {
		t.Errorf("varsayılan timeout: beklenen 2s, alınan %v", cfg.Timeout)
	}
	if cfg.MaxGoroutines != 64 {
		t.Errorf("varsayılan MaxGoroutines: beklenen 64, alınan %d", cfg.MaxGoroutines)
	}
}
