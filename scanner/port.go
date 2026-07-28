// port.go, belirtilen IP adresinin açık portlarını eşzamanlı olarak tarar.
// TCP bağlantısı deneyerek portun açık olup olmadığını kontrol eder.
// Sonuçlar bir channel üzerinden PortResult struct'ı olarak döndürülür.
package scanner

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// PortResult, bir port tarama sonucunu temsil eder.
type PortResult struct {
	Port    int           // Taranan port numarası
	Open    bool          // Port açık mı?
	Service string        // Bilinen servis adı (ör. "HTTP", "SSH")
	Latency time.Duration // Bağlantı süresi
}

// PortScanConfig, port tarayıcı yapılandırmasını tutar.
type PortScanConfig struct {
	Timeout       time.Duration // Her port bağlantısı için zaman aşımı
	MaxGoroutines int           // Eşzamanlı goroutine sayısı
}

// DefaultPortScanConfig, varsayılan port tarama yapılandırmasını döndürür.
func DefaultPortScanConfig() PortScanConfig {
	return PortScanConfig{
		Timeout:       1 * time.Second,
		MaxGoroutines: 128,
	}
}

// CommonPorts, yaygın olarak kullanılan portları döndürür.
func CommonPorts() []int {
	return []int{
		21,    // FTP
		22,    // SSH
		23,    // Telnet
		25,    // SMTP
		53,    // DNS
		80,    // HTTP
		110,   // POP3
		111,   // RPCbind
		135,   // MSRPC
		139,   // NetBIOS
		143,   // IMAP
		443,   // HTTPS
		445,   // SMB
		993,   // IMAPS
		995,   // POP3S
		1433,  // MSSQL
		1521,  // Oracle
		3306,  // MySQL
		3389,  // RDP
		5432,  // PostgreSQL
		5900,  // VNC
		6379,  // Redis
		8080,  // HTTP Alt
		8443,  // HTTPS Alt
		27017, // MongoDB
	}
}

// knownServices, port numarasından servis adını çözümler.
var knownServices = map[int]string{
	21:    "FTP",
	22:    "SSH",
	23:    "Telnet",
	25:    "SMTP",
	53:    "DNS",
	80:    "HTTP",
	110:   "POP3",
	111:   "RPCbind",
	135:   "MSRPC",
	139:   "NetBIOS",
	143:   "IMAP",
	443:   "HTTPS",
	445:   "SMB",
	993:   "IMAPS",
	995:   "POP3S",
	1433:  "MSSQL",
	1521:  "Oracle",
	3306:  "MySQL",
	3389:  "RDP",
	5432:  "PostgreSQL",
	5900:  "VNC",
	6379:  "Redis",
	8080:  "HTTP-Alt",
	8443:  "HTTPS-Alt",
	27017: "MongoDB",
}

// ScanPorts, verilen IP'nin belirtilen portlarını tarar.
// Sonuçlar döndürülen channel üzerinden iletilir; tüm tarama
// bittiğinde channel otomatik olarak kapatılır.
func ScanPorts(ip string, ports []int, cfg PortScanConfig) <-chan PortResult {
	results := make(chan PortResult, len(ports))

	go func() {
		var wg sync.WaitGroup
		sem := make(chan struct{}, cfg.MaxGoroutines)

		for _, port := range ports {
			wg.Add(1)
			sem <- struct{}{}

			go func(p int) {
				defer wg.Done()
				defer func() { <-sem }()

				result := probePort(ip, p, cfg.Timeout)
				results <- result
			}(port)
		}

		wg.Wait()
		close(results)
	}()

	return results
}

// ScanCommonPorts, verilen IP'nin yaygın portlarını varsayılan
// yapılandırma ile tarar.
func ScanCommonPorts(ip string) <-chan PortResult {
	return ScanPorts(ip, CommonPorts(), DefaultPortScanConfig())
}

// ScanPortRange, verilen IP'nin belirtilen port aralığını tarar.
func ScanPortRange(ip string, startPort, endPort int, cfg PortScanConfig) (<-chan PortResult, error) {
	if startPort < 1 || endPort > 65535 || startPort > endPort {
		return nil, fmt.Errorf("geçersiz port aralığı: %d-%d", startPort, endPort)
	}

	ports := make([]int, 0, endPort-startPort+1)
	for p := startPort; p <= endPort; p++ {
		ports = append(ports, p)
	}

	return ScanPorts(ip, ports, cfg), nil
}

// probePort, TCP bağlantısı deneyerek portun açık olup olmadığını kontrol eder.
func probePort(ip string, port int, timeout time.Duration) PortResult {
	result := PortResult{
		Port: port,
		Open: false,
	}

	// Bilinen servis adını ekle
	if svc, ok := knownServices[port]; ok {
		result.Service = svc
	}

	addr := fmt.Sprintf("%s:%d", ip, port)
	start := time.Now()

	conn, err := net.DialTimeout("tcp", addr, timeout)
	result.Latency = time.Since(start)

	if err != nil {
		return result
	}

	conn.Close()
	result.Open = true

	return result
}
