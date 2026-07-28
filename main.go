// BUL — Ağ Tarayıcı uygulamasının giriş noktası.
// scanner paketinden gelen Channel sonuçlarını, ui paketindeki Fyne
// tablosuna Fyne Data Binding aracılığıyla anlık (canlı) olarak yansıtır.
// Tarama işlemi ayrı bir goroutine'de çalışır; UI thread'i hiçbir
// zaman bloklanmaz.
package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"

	"ara/scanner"
	"ara/ui"
)

// ---------- Data Binding Katmanı ----------

type ScanDataStore struct {
	List  binding.UntypedList
	appUI *ui.AppUI
}

func NewScanDataStore(appUI *ui.AppUI) *ScanDataStore {
	store := &ScanDataStore{
		List:  binding.NewUntypedList(),
		appUI: appUI,
	}
	store.List.AddListener(binding.NewDataListener(func() {
		appUI.ResultTable.Refresh()
	}))
	return store
}

func (s *ScanDataStore) Append(row ui.ScanResultRow) {
	s.appUI.AddRow(row)
	_ = s.List.Append(row)
}

func (s *ScanDataStore) Clear() {
	s.appUI.ClearRows()
	s.List.Set([]interface{}{}) //nolint:errcheck
}

// ---------- IP Aralığı ----------

func buildIPRange(startIP, endIP string) (string, error) {
	startIP = strings.TrimSpace(startIP)
	endIP = strings.TrimSpace(endIP)
	if startIP == "" {
		return "", fmt.Errorf("başlangıç IP adresi boş olamaz")
	}
	if endIP == "" {
		return startIP, nil
	}
	startParts := strings.Split(startIP, ".")
	endParts := strings.Split(endIP, ".")
	if len(startParts) != 4 || len(endParts) != 4 {
		return "", fmt.Errorf("geçersiz IP formatı")
	}
	prefix := strings.Join(startParts[:3], ".")
	endPrefix := strings.Join(endParts[:3], ".")
	if prefix != endPrefix {
		return "", fmt.Errorf("başlangıç ve bitiş IP'lerin ilk üç okteti aynı olmalı (%s ≠ %s)", prefix, endPrefix)
	}
	return fmt.Sprintf("%s-%s", startIP, endParts[3]), nil
}

// ---------- Ağ Tarama Orkestrasyon ----------

func startScan(appUI *ui.AppUI, store *ScanDataStore) {
	ipRange, err := buildIPRange(appUI.StartIPEntry.Text, appUI.EndIPEntry.Text)
	if err != nil {
		appUI.SetStatus(fmt.Sprintf("❌ Hata: %s", err.Error()))
		return
	}
	ips, err := scanner.ParseIPRange(ipRange)
	if err != nil {
		appUI.SetStatus(fmt.Sprintf("❌ Hata: %s", err.Error()))
		return
	}
	totalIPs := len(ips)
	appUI.SetScanning(true)
	store.Clear()
	appUI.SetStatus(fmt.Sprintf("⏳ Taranıyor... 0/%d IP", totalIPs))

	resultCh, err := scanner.Scan(ipRange)
	if err != nil {
		appUI.SetStatus(fmt.Sprintf("❌ Tarayıcı hatası: %s", err.Error()))
		appUI.SetScanning(false)
		return
	}

	go func() {
		scanned, activeCount := 0, 0
		scanStart := time.Now()
		for result := range resultCh {
			scanned++
			row := ui.ScanResultRow{
				IP: result.IP, Active: result.Active, MACAddress: result.MACAddress,
			}
			if result.Active {
				activeCount++
				ms := float64(result.ResponseTime.Microseconds()) / 1000.0
				row.ResponseTime = fmt.Sprintf("%.2f ms", ms)
			}
			s, a := scanned, activeCount
			fyne.Do(func() {
				store.Append(row)
				appUI.SetStatus(fmt.Sprintf("⏳ Taranıyor... %d/%d IP  |  %d aktif bulundu", s, totalIPs, a))
			})
		}
		elapsed := time.Since(scanStart)
		fs, fa := scanned, activeCount
		fyne.Do(func() {
			appUI.SetScanning(false)
			appUI.SetStatus(fmt.Sprintf("✅ Tamamlandı — %d/%d IP  |  %d aktif  |  %.1f sn", fs, totalIPs, fa, elapsed.Seconds()))
		})
	}()
}

// ---------- Port Tarama Orkestrasyon ----------

func startPortScan(appUI *ui.AppUI) {
	ip := appUI.SelectedIP
	if ip == "" {
		appUI.SetPortStatus("❌ Lütfen önce bir IP seçin.")
		return
	}

	appUI.SetPortScanning(true)
	appUI.ClearPortRows()

	var resultCh <-chan scanner.PortResult
	var totalPorts int
	customPortStr := strings.TrimSpace(appUI.PortEntry.Text)

	if customPortStr != "" {
		customPortStr = strings.ReplaceAll(customPortStr, ":", "-")
		if strings.Contains(customPortStr, "-") {
			parts := strings.Split(customPortStr, "-")
			if len(parts) != 2 {
				appUI.SetPortStatus("❌ Hatalı port formatı! (Ör: 50-120)")
				appUI.SetPortScanning(false)
				return
			}
			startPort, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			endPort, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 != nil || err2 != nil || startPort < 1 || endPort > 65535 || startPort > endPort {
				appUI.SetPortStatus("❌ Hatalı port aralığı! (1-65535 arası olmalı, Ör: 50-120)")
				appUI.SetPortScanning(false)
				return
			}
			totalPorts = endPort - startPort + 1
			appUI.SetPortStatus(fmt.Sprintf("⏳ %s — Portlar %d-%d taranıyor...", ip, startPort, endPort))
			resultCh, _ = scanner.ScanPortRange(ip, startPort, endPort, scanner.DefaultPortScanConfig())
		} else {
			port, err := strconv.Atoi(customPortStr)
			if err != nil || port < 1 || port > 65535 {
				appUI.SetPortStatus("❌ Hatalı port numarası! (1-65535 arası olmalı)")
				appUI.SetPortScanning(false)
				return
			}
			totalPorts = 1
			appUI.SetPortStatus(fmt.Sprintf("⏳ %s — Port %d taranıyor...", ip, port))
			resultCh = scanner.ScanPorts(ip, []int{port}, scanner.DefaultPortScanConfig())
		}
	} else {
		ports := scanner.CommonPorts()
		totalPorts = len(ports)
		appUI.SetPortStatus(fmt.Sprintf("⏳ %s — Yaygın portlar taranıyor... 0/%d", ip, totalPorts))
		resultCh = scanner.ScanCommonPorts(ip)
	}

	go func() {
		scanned, openCount := 0, 0
		scanStart := time.Now()
		for result := range resultCh {
			scanned++
			row := ui.PortResultRow{
				Port: result.Port, Open: result.Open, Service: result.Service,
			}
			if result.Open {
				openCount++
				ms := float64(result.Latency.Microseconds()) / 1000.0
				row.Latency = fmt.Sprintf("%.2f ms", ms)
			}
			s, o := scanned, openCount
			fyne.Do(func() {
				appUI.AddPortRow(row)
				appUI.SetPortStatus(fmt.Sprintf("⏳ %s — %d/%d port  |  %d açık", ip, s, totalPorts, o))
			})
		}
		elapsed := time.Since(scanStart)
		fs, fo := scanned, openCount
		fyne.Do(func() {
			appUI.SetPortScanning(false)
			appUI.SetPortStatus(fmt.Sprintf("✅ %s — %d/%d port tarandı  |  %d açık  |  %.1f sn", ip, fs, totalPorts, fo, elapsed.Seconds()))
		})
	}()
}

// ---------- Ana Giriş ----------

func main() {
	a := app.NewWithID("com.bul.network-scanner")
	a.Settings().SetTheme(&bulTheme{})

	w := a.NewWindow("BUL — Ağ Tarayıcı")
	w.Resize(fyne.NewSize(900, 650))

	appUI := ui.New()
	store := NewScanDataStore(appUI)

	appUI.OnScan(func() { startScan(appUI, store) })
	appUI.StartIPEntry.OnSubmitted = func(_ string) { startScan(appUI, store) }
	appUI.EndIPEntry.OnSubmitted = func(_ string) { startScan(appUI, store) }
	appUI.OnPortScan(func() { startPortScan(appUI) })

	w.SetContent(appUI.Root)
	w.CenterOnScreen()
	w.ShowAndRun()
}

// ---------- Tema ----------

type bulTheme struct{}

func (t *bulTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}
func (t *bulTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}
func (t *bulTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}
func (t *bulTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
