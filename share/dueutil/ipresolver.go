// Package dueutil due 组件通用工具：解决非 RFC 1918 网卡（Tunngle / Hyper-V 虚拟网段）
// 上 due 自动解析私有 IP 失败导致的 ErrNotFoundIPAddress 启动崩溃。
//
// 在每个 main.go 里空导入本包即可自动注册：
//
//	import _ "server-cluster/share/dueutil"
package dueutil

import (
	"net"

	xnet "github.com/dobyte/due/v2/core/net"
)

func init() {
	xnet.SetPrivateIPResolver(resolvePrivateIP)
}

// resolvePrivateIP due 默认的 defaultPrivateIPResolver 只认 RFC 1918 三段私有 IP
// （10/8、172.16/12、192.168/16），在 Tunngle、ZeroTier 等虚拟网卡上会返回 ErrNotFoundIPAddress。
//
// 本实现先沿用默认解析器（优先选 192.168.x.x，其次选任意 RFC 1918），失败则回退到
// 第一个活动网卡的 IPv4（不强制 IsPrivate()），单机开发场景下保证能拿到一个可用地址。
func resolvePrivateIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1", nil
	}

	var firstIPv4 string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ipv4 := ip.To4(); ipv4 != nil {
				s := ipv4.String()
				// 优先返回 192.168 段（局域网最常见）
				if ipv4[0] == 192 && ipv4[1] == 168 {
					return s, nil
				}
				// 其次返回任意 RFC 1918 私有段
				if ipv4.IsPrivate() {
					return s, nil
				}
				// 缓存第一个非 loopback 的 IPv4 作为兜底
				if firstIPv4 == "" {
					firstIPv4 = s
				}
			}
		}
	}

	if firstIPv4 != "" {
		return firstIPv4, nil
	}
	return "127.0.0.1", nil
}
