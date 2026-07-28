// Package ui, Fyne v2 tabanlı minimalist ağ tarayıcı arayüzünü tanımlar.
package ui

import (
	"fmt"
	"image/color"
	"sort"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ---------- Renk Paleti ----------

var (
	colorBg       = color.NRGBA{R: 18, G: 18, B: 24, A: 255}
	colorAccent   = color.NRGBA{R: 0, G: 200, B: 150, A: 255}
	colorTextPri  = color.NRGBA{R: 240, G: 240, B: 245, A: 255}
	colorTextSec  = color.NRGBA{R: 160, G: 165, B: 180, A: 255}
	colorHeaderBg = color.NRGBA{R: 35, G: 35, B: 50, A: 255}
	colorRowEven  = color.NRGBA{R: 24, G: 24, B: 34, A: 255}
	colorRowOdd   = color.NRGBA{R: 30, G: 30, B: 42, A: 255}
	colorOnline   = color.NRGBA{R: 0, G: 220, B: 130, A: 255}
	colorOffline  = color.NRGBA{R: 100, G: 100, B: 120, A: 255}
	colorPortOpen = color.NRGBA{R: 80, G: 200, B: 255, A: 255}
)

// ---------- Veri Modelleri ----------

type ScanResultRow struct {
	IP, ResponseTime, MACAddress string
	Active                       bool
}

type PortResultRow struct {
	Port    int
	Open    bool
	Service string
	Latency string
}

// ---------- AppUI ----------

var columnHeaders = []string{"Durum", "IP Adresi", "Yanıt Süresi", "MAC Adresi"}
var columnWidths = []float32{80, 200, 160, 220}
var portHeaders = []string{"Port", "Durum", "Servis", "Gecikme"}
var portWidths = []float32{100, 100, 160, 140}

type AppUI struct {
	// Sekme 1: Ağ Tarama
	StartIPEntry *widget.Entry
	EndIPEntry   *widget.Entry
	ScanButton   *widget.Button
	StatusLabel  *widget.Label
	ResultTable  *widget.Table
	mu           sync.RWMutex
	rows         []ScanResultRow

	// Sekme 2: Port Tarama
	ActiveIPList    *widget.List
	PortEntry       *widget.Entry
	PortScanButton  *widget.Button
	PortStatusLabel *widget.Label
	PortTable       *widget.Table
	SelectedIP      string
	portMu          sync.RWMutex
	portRows        []PortResultRow
	activeIPs       []string

	// Sekmeler
	Tabs *container.AppTabs
	Root *fyne.Container
}

func New() *AppUI {
	ui := &AppUI{
		rows:     make([]ScanResultRow, 0),
		portRows: make([]PortResultRow, 0),
		activeIPs: make([]string, 0),
	}
	ui.buildEntries()
	ui.buildButtons()
	ui.buildStatusBar()
	ui.buildTable()
	ui.buildPortTab()
	ui.buildLayout()
	return ui
}

// ---------- Sekme 1 Bileşenleri ----------

func (ui *AppUI) buildEntries() {
	ui.StartIPEntry = widget.NewEntry()
	ui.StartIPEntry.SetPlaceHolder("Başlangıç IP  (ör. 192.168.1.1)")
	ui.StartIPEntry.TextStyle = fyne.TextStyle{Monospace: true}
	ui.EndIPEntry = widget.NewEntry()
	ui.EndIPEntry.SetPlaceHolder("Bitiş IP  (ör. 192.168.1.255)")
	ui.EndIPEntry.TextStyle = fyne.TextStyle{Monospace: true}
}

func (ui *AppUI) buildButtons() {
	ui.ScanButton = widget.NewButtonWithIcon("  Tara  ", theme.SearchIcon(), nil)
	ui.ScanButton.Importance = widget.HighImportance
}

func (ui *AppUI) buildStatusBar() {
	ui.StatusLabel = widget.NewLabel("Tarama başlatmak için IP aralığı girin.")
	ui.StatusLabel.TextStyle = fyne.TextStyle{Italic: true}
}

func (ui *AppUI) buildTable() {
	ui.ResultTable = widget.NewTable(
		func() (int, int) {
			ui.mu.RLock()
			defer ui.mu.RUnlock()
			return len(ui.rows), len(columnHeaders)
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(colorRowEven)
			bg.SetMinSize(fyne.NewSize(100, 36))
			label := canvas.NewText("placeholder", colorTextPri)
			label.TextSize = 13
			label.TextStyle = fyne.TextStyle{Monospace: true}
			dot := canvas.NewCircle(colorOnline)
			dot.Resize(fyne.NewSize(10, 10))
			dot.Hide()
			return container.NewStack(bg, container.NewPadded(container.NewHBox(dot, label)))
		},
		func(id widget.TableCellID, tmpl fyne.CanvasObject) {
			ui.mu.RLock()
			defer ui.mu.RUnlock()
			stack := tmpl.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			hbox := stack.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container)
			dot := hbox.Objects[0].(*canvas.Circle)
			label := hbox.Objects[1].(*canvas.Text)
			if id.Row >= len(ui.rows) {
				label.Text = ""
				dot.Hide()
				label.Refresh()
				return
			}
			row := ui.rows[id.Row]
			if id.Row%2 == 0 {
				bg.FillColor = colorRowEven
			} else {
				bg.FillColor = colorRowOdd
			}
			bg.Refresh()
			dot.Hide()
			label.Color = colorTextPri
			switch id.Col {
			case 0:
				dot.Show()
				if row.Active {
					dot.FillColor = colorOnline
					label.Text = " Aktif"
					label.Color = colorOnline
				} else {
					dot.FillColor = colorOffline
					label.Text = " Pasif"
					label.Color = colorOffline
				}
			case 1:
				label.Text = row.IP
			case 2:
				if row.Active {
					label.Text = row.ResponseTime
					label.Color = colorAccent
				} else {
					label.Text = "—"
					label.Color = colorTextSec
				}
			case 3:
				if row.MACAddress != "" {
					label.Text = row.MACAddress
				} else {
					label.Text = "—"
					label.Color = colorTextSec
				}
			}
			dot.Refresh()
			label.Refresh()
		},
	)
	ui.ResultTable.ShowHeaderRow = true
	ui.ResultTable.CreateHeader = createHeader
	ui.ResultTable.UpdateHeader = func(id widget.TableCellID, tmpl fyne.CanvasObject) {
		updateHeader(id, tmpl, columnHeaders)
	}
	for i, w := range columnWidths {
		ui.ResultTable.SetColumnWidth(i, w)
	}
}

// ---------- Sekme 2: Port Tarama Bileşenleri ----------

func (ui *AppUI) buildPortTab() {
	// Aktif IP listesi
	ui.ActiveIPList = widget.NewList(
		func() int {
			ui.mu.RLock()
			defer ui.mu.RUnlock()
			return len(ui.activeIPs)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("placeholder")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			ui.mu.RLock()
			defer ui.mu.RUnlock()
			if id < len(ui.activeIPs) {
				obj.(*widget.Label).SetText("  ● " + ui.activeIPs[id])
			}
		},
	)
	ui.ActiveIPList.OnSelected = func(id widget.ListItemID) {
		ui.mu.RLock()
		defer ui.mu.RUnlock()
		if id < len(ui.activeIPs) {
			ui.SelectedIP = ui.activeIPs[id]
			ui.PortScanButton.SetText("  " + ui.SelectedIP + " → Port Tara  ")
			ui.PortScanButton.Enable()
		}
	}

	// Port giriş alanı ve tarama butonu
	ui.PortEntry = widget.NewEntry()
	ui.PortEntry.SetPlaceHolder("Port (Boş=Yaygın, Ör: 80, 50-120)")
	ui.PortEntry.TextStyle = fyne.TextStyle{Monospace: true}

	ui.PortScanButton = widget.NewButtonWithIcon("  IP seçin  ", theme.SearchIcon(), nil)
	ui.PortScanButton.Importance = widget.HighImportance
	ui.PortScanButton.Disable()

	// Port durum etiketi
	ui.PortStatusLabel = widget.NewLabel("Aktif bir IP seçip port taraması başlatın.")
	ui.PortStatusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Port sonuç tablosu
	ui.PortTable = widget.NewTable(
		func() (int, int) {
			ui.portMu.RLock()
			defer ui.portMu.RUnlock()
			return len(ui.portRows), len(portHeaders)
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(colorRowEven)
			bg.SetMinSize(fyne.NewSize(80, 34))
			label := canvas.NewText("placeholder", colorTextPri)
			label.TextSize = 13
			label.TextStyle = fyne.TextStyle{Monospace: true}
			return container.NewStack(bg, container.NewPadded(label))
		},
		func(id widget.TableCellID, tmpl fyne.CanvasObject) {
			ui.portMu.RLock()
			defer ui.portMu.RUnlock()
			stack := tmpl.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			label := stack.Objects[1].(*fyne.Container).Objects[0].(*canvas.Text)
			if id.Row >= len(ui.portRows) {
				label.Text = ""
				label.Refresh()
				return
			}
			row := ui.portRows[id.Row]
			if id.Row%2 == 0 {
				bg.FillColor = colorRowEven
			} else {
				bg.FillColor = colorRowOdd
			}
			bg.Refresh()
			label.Color = colorTextPri
			switch id.Col {
			case 0:
				label.Text = fmt.Sprintf("%d", row.Port)
			case 1:
				if row.Open {
					label.Text = "● Açık"
					label.Color = colorPortOpen
				} else {
					label.Text = "● Kapalı"
					label.Color = colorOffline
				}
			case 2:
				if row.Service != "" {
					label.Text = row.Service
					if row.Open {
						label.Color = colorAccent
					}
				} else {
					label.Text = "—"
					label.Color = colorTextSec
				}
			case 3:
				if row.Open {
					label.Text = row.Latency
					label.Color = colorPortOpen
				} else {
					label.Text = "—"
					label.Color = colorTextSec
				}
			}
			label.Refresh()
		},
	)
	ui.PortTable.ShowHeaderRow = true
	ui.PortTable.CreateHeader = createHeader
	ui.PortTable.UpdateHeader = func(id widget.TableCellID, tmpl fyne.CanvasObject) {
		updateHeader(id, tmpl, portHeaders)
	}
	for i, w := range portWidths {
		ui.PortTable.SetColumnWidth(i, w)
	}
}

// ---------- Ortak Header Fonksiyonları ----------

func createHeader() fyne.CanvasObject {
	bg := canvas.NewRectangle(colorHeaderBg)
	bg.SetMinSize(fyne.NewSize(80, 34))
	label := canvas.NewText("Header", colorTextPri)
	label.TextSize = 13
	label.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewStack(bg, container.NewPadded(label))
}

func updateHeader(id widget.TableCellID, tmpl fyne.CanvasObject, headers []string) {
	stack := tmpl.(*fyne.Container)
	bg := stack.Objects[0].(*canvas.Rectangle)
	label := stack.Objects[1].(*fyne.Container).Objects[0].(*canvas.Text)
	bg.FillColor = colorHeaderBg
	bg.Refresh()
	if id.Col >= 0 && id.Col < len(headers) {
		label.Text = headers[id.Col]
		label.Color = colorAccent
		label.TextStyle = fyne.TextStyle{Bold: true}
	}
	label.Refresh()
}

// ---------- Düzen ----------

func (ui *AppUI) buildLayout() {
	// Başlık
	titleText := canvas.NewText("⟐  BUL — Ağ Tarayıcı", colorTextPri)
	titleText.TextSize = 20
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	subtitleText := canvas.NewText("Yerel ağınızdaki aktif cihazları keşfedin", colorTextSec)
	subtitleText.TextSize = 12
	titleSection := container.NewVBox(titleText, subtitleText, widget.NewSeparator())

	// Giriş paneli
	startLabel := canvas.NewText("Başlangıç", colorTextSec)
	startLabel.TextSize = 11
	startLabel.TextStyle = fyne.TextStyle{Bold: true}
	endLabel := canvas.NewText("Bitiş", colorTextSec)
	endLabel.TextSize = 11
	endLabel.TextStyle = fyne.TextStyle{Bold: true}
	inputRow := container.NewHBox(
		container.New(layout.NewGridWrapLayout(fyne.NewSize(260, 68)), container.NewVBox(startLabel, ui.StartIPEntry)),
		container.New(layout.NewGridWrapLayout(fyne.NewSize(260, 68)), container.NewVBox(endLabel, ui.EndIPEntry)),
		layout.NewSpacer(),
		container.NewVBox(canvas.NewText(" ", color.Transparent), ui.ScanButton),
	)
	statusRow := container.NewHBox(canvas.NewText("◉", colorAccent), ui.StatusLabel, layout.NewSpacer())
	topSection := container.NewVBox(titleSection, inputRow, widget.NewSeparator(), statusRow)

	// Sekme 1: Ağ tarama
	tab1Content := container.NewBorder(topSection, nil, nil, nil, ui.ResultTable)

	// Sekme 2: Port tarama
	ipListLabel := canvas.NewText("Aktif Cihazlar", colorAccent)
	ipListLabel.TextSize = 14
	ipListLabel.TextStyle = fyne.TextStyle{Bold: true}
	portStatusRow := container.NewHBox(canvas.NewText("◉", colorPortOpen), ui.PortStatusLabel, layout.NewSpacer())
	
	portControlRow := container.NewHBox(
		container.New(layout.NewGridWrapLayout(fyne.NewSize(160, 36)), ui.PortEntry),
		ui.PortScanButton,
	)

	leftPanel := container.NewBorder(
		container.NewVBox(ipListLabel, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), portControlRow),
		nil, nil,
		ui.ActiveIPList,
	)
	rightPanel := container.NewBorder(portStatusRow, nil, nil, nil, ui.PortTable)
	tab2Content := container.NewHSplit(leftPanel, rightPanel)
	tab2Content.SetOffset(0.3)

	// Sekmeler
	ui.Tabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Ağ Tarama", theme.ComputerIcon(), tab1Content),
		container.NewTabItemWithIcon("Port Tarama", theme.SearchIcon(), tab2Content),
	)
	ui.Tabs.SetTabLocation(container.TabLocationTop)

	bgRect := canvas.NewRectangle(colorBg)
	ui.Root = container.NewStack(bgRect, container.NewPadded(ui.Tabs))
}

// ---------- Ağ Tarama Veri Yönetimi ----------

func (ui *AppUI) AddRow(row ScanResultRow) {
	ui.mu.Lock()
	ui.rows = append(ui.rows, row)
	if row.Active {
		ui.activeIPs = append(ui.activeIPs, row.IP)
	}
	ui.mu.Unlock()
	ui.ResultTable.Refresh()
	ui.ActiveIPList.Refresh()
}

func (ui *AppUI) ClearRows() {
	ui.mu.Lock()
	ui.rows = ui.rows[:0]
	ui.activeIPs = ui.activeIPs[:0]
	ui.mu.Unlock()
	ui.SelectedIP = ""
	ui.PortScanButton.SetText("  IP seçin  ")
	ui.PortScanButton.Disable()
	ui.ResultTable.Refresh()
	ui.ActiveIPList.Refresh()
}

func (ui *AppUI) SetRows(rows []ScanResultRow) {
	ui.mu.Lock()
	ui.rows = rows
	ui.activeIPs = ui.activeIPs[:0]
	for _, r := range rows {
		if r.Active {
			ui.activeIPs = append(ui.activeIPs, r.IP)
		}
	}
	ui.mu.Unlock()
	ui.ResultTable.Refresh()
	ui.ActiveIPList.Refresh()
}

func (ui *AppUI) RowCount() int {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	return len(ui.rows)
}

func (ui *AppUI) GetRows() []ScanResultRow {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	copied := make([]ScanResultRow, len(ui.rows))
	copy(copied, ui.rows)
	return copied
}

func (ui *AppUI) SetStatus(text string)  { ui.StatusLabel.SetText(text) }
func (ui *AppUI) OnScan(fn func())       { ui.ScanButton.OnTapped = fn }
func (ui *AppUI) OnPortScan(fn func())   { ui.PortScanButton.OnTapped = fn }

func (ui *AppUI) SetScanning(scanning bool) {
	if scanning {
		ui.ScanButton.Disable()
		ui.StartIPEntry.Disable()
		ui.EndIPEntry.Disable()
		ui.ScanButton.SetText("  Taranıyor...  ")
	} else {
		ui.ScanButton.Enable()
		ui.StartIPEntry.Enable()
		ui.EndIPEntry.Enable()
		ui.ScanButton.SetText("  Tara  ")
	}
}

// ---------- Port Tarama Veri Yönetimi ----------

func (ui *AppUI) AddPortRow(row PortResultRow) {
	ui.portMu.Lock()
	ui.portRows = append(ui.portRows, row)
	
	// Açık portlar üstte, sonrasında port numarasına göre sırala
	sort.Slice(ui.portRows, func(i, j int) bool {
		if ui.portRows[i].Open != ui.portRows[j].Open {
			return ui.portRows[i].Open // true olan (açık port) öne gelir
		}
		return ui.portRows[i].Port < ui.portRows[j].Port // Aynı durumdalarsa küçük port öne gelir
	})
	
	ui.portMu.Unlock()
	ui.PortTable.Refresh()
}

func (ui *AppUI) ClearPortRows() {
	ui.portMu.Lock()
	ui.portRows = ui.portRows[:0]
	ui.portMu.Unlock()
	ui.PortTable.Refresh()
}

func (ui *AppUI) SetPortStatus(text string) { ui.PortStatusLabel.SetText(text) }

func (ui *AppUI) SetPortScanning(scanning bool) {
	if scanning {
		ui.PortScanButton.Disable()
		ui.PortEntry.Disable()
		ui.PortScanButton.SetText("  Taranıyor...  ")
	} else {
		ui.PortScanButton.Enable()
		ui.PortEntry.Enable()
		if ui.SelectedIP != "" {
			ui.PortScanButton.SetText("  " + ui.SelectedIP + " → Port Tara  ")
		} else {
			ui.PortScanButton.SetText("  IP seçin  ")
		}
	}
}

func (ui *AppUI) PortRowCount() int {
	ui.portMu.RLock()
	defer ui.portMu.RUnlock()
	return len(ui.portRows)
}

// SwitchToPortTab, port tarama sekmesine geçer.
func (ui *AppUI) SwitchToPortTab() {
	ui.Tabs.SelectIndex(1)
}
