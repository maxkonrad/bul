# BUL — Network & Port Scanner 🔍

**BUL** is a concurrent local network and port scanner that combines the power of Go (Goroutines and Channels) with a modern, minimalist graphical user interface built on Fyne v2. With its user-friendly, high-contrast dark theme, it allows you to discover and analyze active devices on your network in seconds.

## Screenshots

### Network Scan (Device Discovery)
![Network Scan](screenshots/network_scan.png)
*By specifying an IP range, it concurrently detects the status, response times (ms), and MAC addresses of active devices on the network.*

### Port Scan (Service Detection)
![Port Scan](screenshots/port_scan.png)
*Scans the top 25 common ports, a specific port (e.g., 8080), or a specific port range (e.g., 50-120) of selected active devices, automatically highlighting and sorting open ports at the top.*

---

## Features ✨

- **Fast Network Scanning:** Scans the provided IP range (e.g., `192.168.1.1` - `255`) using ICMP (Ping) to report whether the device is active or inactive.
- **Detailed Information:** Resolves response times (RTT) and MAC addresses (via the ARP table) for active devices.
- **Advanced Port Scanning:** Select any discovered device from the list and perform a port scan with a single click.
  - **Common Ports:** If left blank, it automatically scans 25 critical ports (HTTP, HTTPS, SSH, FTP, MySQL, etc.).
  - **Custom/Range Scanning:** You can specify a single port (e.g., `8080`) or a port range (e.g., `50-120`).
  - **Smart Sorting:** Open ports are automatically moved to the top of the list once the scan finishes.
- **Concurrency & Performance:** Uses `Goroutines`, `Channels`, and Semaphore architecture in the background for maximum efficiency without freezing the UI. Thanks to Fyne's Data Binding, results are rendered in real-time.
- **Cross-Platform:** Can be compiled for Windows (.exe), Linux, and macOS (Apple Silicon & Intel) using GitHub Actions, functioning entirely as a native desktop application.

## Usage 🚀

### Running as a Developer
Make sure you have [Go 1.22+](https://go.dev/) installed on your system.
```bash
# Download dependencies
go mod tidy

# Run the application (sudo may be required for ICMP Ping packets)
go run .
```

### Downloading Pre-compiled Binaries
You can navigate to the **Releases** tab on the project's GitHub repository to download the executable suitable for your operating system and run it directly without any installation.

## Tech Stack 💻
- **Backend:** Go (Golang)
- **Concurrency:** Goroutines, Channels, sync.WaitGroup, sync.RWMutex
- **GUI:** [Fyne v2](https://fyne.io/)
- **Data Binding:** `fyne.io/fyne/v2/data/binding` 
- **System Communication:** `os/exec` for ARP table reading and ICMP ping execution.

---
*Developed by Furkan Selek.*
