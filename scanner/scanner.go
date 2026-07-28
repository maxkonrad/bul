// Package scanner, verilen bir IP aralığını eşzamanlı olarak tarar.
// Her IP adresine ICMP ping atarak aktif olup olmadığını kontrol eder,
// yanıt süresini ölçer ve mümkünse MAC adresini çözümler.
// Sonuçlar bir channel üzerinden ScanResult struct'ı olarak döndürülür.
package scanner

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ScanResult, bir IP tarama sonucunu temsil eder.
type ScanResult struct {
	IP           string        // Taranan IP adresi (ör. "192.168.1.1")
	Active       bool          // IP'nin aktif (erişilebilir) olup olmadığı
	ResponseTime time.Duration // Ping yanıt süresi
	MACAddress   string        // ARP tablosundan çözümlenen MAC adresi (bulunamazsa boş)
}

// ScanConfig, tarayıcı yapılandırmasını tutar.
type ScanConfig struct {
	Timeout       time.Duration // Her ping işlemi için zaman aşımı süresi
	MaxGoroutines int           // Aynı anda çalışacak maksimum goroutine sayısı
}

// DefaultConfig, varsayılan tarayıcı yapılandırmasını döndürür.
func DefaultConfig() ScanConfig {
	return ScanConfig{
		Timeout:       2 * time.Second,
		MaxGoroutines: 64,
	}
}

// ParseIPRange, "192.168.1.1-255" gibi bir IP aralığı ifadesini ayrıştırarak
// taranacak tüm IP adreslerinin listesini döndürür.
// Desteklenen formatlar:
//   - "192.168.1.1-255"   → 192.168.1.1 ile 192.168.1.255 arası
//   - "192.168.1.10-20"   → 192.168.1.10 ile 192.168.1.20 arası
//   - "192.168.1.1"       → Yalnızca tek IP
func ParseIPRange(ipRange string) ([]string, error) {
	// Tire ile ayrılmış bitiş değeri var mı kontrol et
	parts := strings.SplitN(ipRange, "-", 2)

	basePart := strings.TrimSpace(parts[0])
	baseOctets := strings.Split(basePart, ".")
	if len(baseOctets) != 4 {
		return nil, fmt.Errorf("geçersiz IP formatı: %s (4 oktet gerekli)", basePart)
	}

	// Her okteti doğrula
	octets := make([]int, 4)
	for i, o := range baseOctets {
		val, err := strconv.Atoi(strings.TrimSpace(o))
		if err != nil || val < 0 || val > 255 {
			return nil, fmt.Errorf("geçersiz oktet değeri: %s", o)
		}
		octets[i] = val
	}

	startLast := octets[3]
	endLast := startLast

	// Eğer aralık belirtilmişse bitiş değerini al
	if len(parts) == 2 {
		end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || end < 0 || end > 255 {
			return nil, fmt.Errorf("geçersiz aralık bitiş değeri: %s", parts[1])
		}
		endLast = end
	}

	if startLast > endLast {
		return nil, fmt.Errorf("başlangıç (%d) bitiş değerinden (%d) büyük olamaz", startLast, endLast)
	}

	ips := make([]string, 0, endLast-startLast+1)
	for i := startLast; i <= endLast; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d", octets[0], octets[1], octets[2], i)
		ips = append(ips, ip)
	}

	return ips, nil
}

// Scan, verilen IP aralığını varsayılan yapılandırma ile tarar.
// Sonuçlar döndürülen channel üzerinden iletilir; tüm tarama
// bittiğinde channel otomatik olarak kapatılır.
func Scan(ipRange string) (<-chan ScanResult, error) {
	return ScanWithConfig(ipRange, DefaultConfig())
}

// ScanWithConfig, verilen IP aralığını belirtilen yapılandırma ile tarar.
// Sonuçlar döndürülen channel üzerinden iletilir; tüm tarama
// bittiğinde channel otomatik olarak kapatılır.
func ScanWithConfig(ipRange string, cfg ScanConfig) (<-chan ScanResult, error) {
	ips, err := ParseIPRange(ipRange)
	if err != nil {
		return nil, fmt.Errorf("IP aralığı ayrıştırılamadı: %w", err)
	}

	results := make(chan ScanResult, len(ips))

	go func() {
		var wg sync.WaitGroup
		// Eşzamanlılığı sınırlamak için semaphore pattern
		sem := make(chan struct{}, cfg.MaxGoroutines)

		for _, ip := range ips {
			wg.Add(1)
			sem <- struct{}{} // Slot al

			go func(targetIP string) {
				defer wg.Done()
				defer func() { <-sem }() // Slotu serbest bırak

				result := pingHost(targetIP, cfg.Timeout)
				results <- result
			}(ip)
		}

		wg.Wait()
		close(results)
	}()

	return results, nil
}

// ScanIPs, verilen IP listesini doğrudan tarar.
// IP aralığı yerine belirli IP'ler taranmak istendiğinde kullanılır.
func ScanIPs(ips []string, cfg ScanConfig) <-chan ScanResult {
	results := make(chan ScanResult, len(ips))

	go func() {
		var wg sync.WaitGroup
		sem := make(chan struct{}, cfg.MaxGoroutines)

		for _, ip := range ips {
			wg.Add(1)
			sem <- struct{}{}

			go func(targetIP string) {
				defer wg.Done()
				defer func() { <-sem }()

				result := pingHost(targetIP, cfg.Timeout)
				results <- result
			}(ip)
		}

		wg.Wait()
		close(results)
	}()

	return results
}

// pingHost, belirtilen IP adresine ICMP ping atar ve sonucu döndürür.
// Linux'ta "ping -c 1 -W <timeout>" komutu kullanılır.
func pingHost(ip string, timeout time.Duration) ScanResult {
	result := ScanResult{
		IP:     ip,
		Active: false,
	}

	timeoutSec := int(timeout.Seconds())
	if timeoutSec < 1 {
		timeoutSec = 1
	}

	start := time.Now()

	// ICMP ping komutu (Linux)
	cmd := exec.Command("ping", "-c", "1", "-W", strconv.Itoa(timeoutSec), ip)
	output, err := cmd.CombinedOutput()
	elapsed := time.Since(start)

	if err != nil {
		// Ping başarısız — host aktif değil
		return result
	}

	result.Active = true
	result.ResponseTime = elapsed

	// Ping çıktısından gerçek RTT değerini çözümlemeye çalış
	// Örnek: "time=1.23 ms" veya "time=0.456 ms"
	if rtt := extractRTT(string(output)); rtt > 0 {
		result.ResponseTime = rtt
	}

	// ARP tablosundan MAC adresini çözümle
	result.MACAddress = lookupMAC(ip)

	return result
}

// extractRTT, ping çıktısından "time=X.XX ms" değerini ayrıştırır.
func extractRTT(output string) time.Duration {
	re := regexp.MustCompile(`time[=<]([\d.]+)\s*ms`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0
	}

	ms, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	return time.Duration(ms*1000) * time.Microsecond
}

// lookupMAC, ARP tablosunu sorgulayarak verilen IP'nin MAC adresini döndürür.
// MAC adresi bulunamazsa boş string döner.
func lookupMAC(ip string) string {
	// Önce Go'nun net paketini kullanarak deneme
	if mac := lookupMACFromInterfaces(ip); mac != "" {
		return mac
	}

	// ARP tablosunu komut satırından sorgula
	return lookupMACFromARP(ip)
}

// lookupMACFromInterfaces, yerel ağ arayüzlerini kontrol ederek
// verilen IP'nin bu makinedeyse MAC adresini döndürür.
func lookupMACFromInterfaces(ip string) string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ifaceIP net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ifaceIP = v.IP
			case *net.IPAddr:
				ifaceIP = v.IP
			}
			if ifaceIP != nil && ifaceIP.String() == ip {
				return iface.HardwareAddr.String()
			}
		}
	}

	return ""
}

// lookupMACFromARP, Linux ARP tablosunu sorgulayarak MAC adresini döndürür.
// /proc/net/arp dosyasını veya "ip neigh" komutunu kullanır.
func lookupMACFromARP(ip string) string {
	// Önce ip neigh komutunu dene (modern Linux)
	cmd := exec.Command("ip", "neigh", "show", ip)
	output, err := cmd.Output()
	if err == nil {
		mac := parseIPNeighOutput(string(output))
		if mac != "" {
			return mac
		}
	}

	// Alternatif: arp komutunu dene
	cmd = exec.Command("arp", "-n", ip)
	output, err = cmd.Output()
	if err == nil {
		mac := parseARPOutput(string(output))
		if mac != "" {
			return mac
		}
	}

	return ""
}

// parseIPNeighOutput, "ip neigh show" komut çıktısından MAC adresini ayrıştırır.
// Örnek çıktı: "192.168.1.1 dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE"
func parseIPNeighOutput(output string) string {
	re := regexp.MustCompile(`lladdr\s+([0-9a-fA-F:]{17})`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return strings.ToUpper(matches[1])
	}
	return ""
}

// parseARPOutput, "arp -n" komut çıktısından MAC adresini ayrıştırır.
// Örnek çıktı:
// Address         HWtype  HWaddress           Flags Mask  Iface
// 192.168.1.1     ether   aa:bb:cc:dd:ee:ff   C           eth0
func parseARPOutput(output string) string {
	re := regexp.MustCompile(`([0-9a-fA-F]{2}(?::[0-9a-fA-F]{2}){5})`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 1 {
		return strings.ToUpper(matches[0])
	}
	return ""
}
