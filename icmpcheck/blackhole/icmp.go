package main
import (
	"syscall"
	"fmt"
	"os"
)

type icmpSender struct {
	icmpFd int
	icmpv6Fd int
	mtu int
}

func NewIcmpSender(mtu int) *icmpSender{
	is := &icmpSender{}
	is.mtu = mtu


	icmpFd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	if err != nil {
		fmt.Fprintf(os.Stderr, "socket(AF_INET, SOCK_RAW, IPPROTO_ICMP): %s\n", err)
		os.Exit(1)
	}

	syscall.SetsockoptInt(icmpFd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 0)
	syscall.SetsockoptInt(icmpFd, syscall.SOL_SOCKET, syscall.SO_MARK, 33)

	icmpv6Fd, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_ICMPV6)
	if err != nil {
		fmt.Fprintf(os.Stderr, "socket(AF_INET6, SOCK_RAW, IPPROTO_ICMPV6): %s\n", err)
		os.Exit(1)
	}

	syscall.SetsockoptInt(icmpv6Fd, syscall.IPPROTO_IPV6, syscall.IP_HDRINCL, 0)
	syscall.SetsockoptInt(icmpv6Fd, syscall.SOL_SOCKET, syscall.SO_MARK, 33)

	is.icmpFd = icmpFd
	is.icmpv6Fd = icmpv6Fd
	return is
}

func (is *icmpSender)replyIcmp(p *Packet) {
	mtu := is.mtu
	var icmpData []byte

	truncData := p.L3Data
	if len(truncData) > 64 {
		truncData = truncData[:64]
	}

	if p.IP.Ipver == 4 {
		icmpData = []byte {
			3, 4, 0, 0,
			byte((mtu >> 24) & 0xFF), byte((mtu >> 16) & 0xFF), byte((mtu >> 8) & 0xFF), byte((mtu >> 0) & 0xFF),
		}
		icmpData = append(icmpData, truncData...)
		cs := checksum(icmpData)
		icmpData[2] = uint8(cs >> 8)
		icmpData[3] = uint8(cs)

		var r [4] byte
		copy(r[:], p.IP.SrcIP.To4())
		addr := syscall.SockaddrInet4{
			Port: 0,
			Addr: r,
		}

		err := syscall.Sendto(is.icmpFd, icmpData, 0, &addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sendto %s\n", err)
		}
	}

	if p.IP.Ipver == 6 {
		mtu = 1285
		icmpData = []byte {
			2, 0, 0, 0,
			byte((mtu >> 24) & 0xFF), byte((mtu >> 16) & 0xFF), byte((mtu >> 8) & 0xFF), byte((mtu >> 0) & 0xFF),
		}
		icmpData = append(icmpData, truncData...)

		// TODO: Leaving checksum to zero is almost certainly not going to work. IPv6 wonders!

		var r [16] byte
		copy(r[:], p.IP.SrcIP.To16())
		addr := syscall.SockaddrInet6{
			Port: 0,
			Addr: r,
		}

		err := syscall.Sendto(is.icmpv6Fd, icmpData, 0, &addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sendto %s\n", err)
		}
	}
}
