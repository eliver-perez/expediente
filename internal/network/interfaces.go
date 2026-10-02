package network

import (
	"net"
	"net/netip"
	"sort"
	"strings"
)

type Interface struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	URL       string `json:"url"`
	Suggested bool   `json:"suggested"`
}

func virtualInterface(name string) bool {
	name = strings.ToLower(name)
	for _, prefix := range []string{"docker", "veth", "virbr", "br-", "bridge", "vmnet", "vboxnet", "vethernet", "utun", "tun", "tap", "tailscale", "wg", "zt", "awdl", "llw"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return strings.Contains(name, "vpn") || strings.Contains(name, "virtual")
}

// Keep a VM's primary Ethernet interface: it is the user's LAN interface inside
// that guest. Only known tunnel/bridge names are de-emphasized in diagnostics.
func interfaces(port string) ([]Interface, error) {
	devices, err := net.Interfaces()
	result := []Interface{}
	if err != nil {
		return result, err
	}
	for _, device := range devices {
		if device.Flags&net.FlagUp == 0 || device.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := device.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			prefix, err := netip.ParsePrefix(address.String())
			if err != nil {
				continue
			}
			ip := prefix.Addr().Unmap()
			if !ip.Is4() || !ip.IsGlobalUnicast() || ip.IsLinkLocalUnicast() || ip.IsLoopback() {
				continue
			}
			result = append(result, Interface{Name: device.Name, Address: ip.String(), URL: "http://" + net.JoinHostPort(ip.String(), port), Suggested: !virtualInterface(device.Name)})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Suggested != result[j].Suggested {
			return result[i].Suggested
		}
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].Address < result[j].Address
	})
	return result, nil
}

// HTTP LAN accepts numeric addresses actually assigned to this machine, never
// arbitrary Host headers or forwarded headers. This also prevents DNS rebinding.
func AllowsHost(host, listenAddress string) bool {
	address, port, err := net.SplitHostPort(host)
	_, expectedPort, _ := net.SplitHostPort(listenAddress)
	if err != nil || port != expectedPort {
		return false
	}
	if address == "127.0.0.1" {
		return true
	}
	ip, err := netip.ParseAddr(address)
	if err != nil || !ip.Is4() || ip.String() != address {
		return false
	}
	available, err := interfaces(port)
	if err != nil {
		return false
	}
	for _, device := range available {
		if device.Address == address {
			return true
		}
	}
	return false
}
