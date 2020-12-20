package util

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"

	"github.com/mitchellh/go-ps"
	"github.com/tinyzimmer/k3p/pkg/log"
)

// RevBytes will reverse the given byte slice.
func RevBytes(b []byte) []byte {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return b
}

// RevIPv4 will reverse an IPv4 address
func RevIPv4(ip net.IP) net.IP {
	rev := RevBytes(ip.To4())
	return net.IPv4(rev[0], rev[1], rev[2], rev[3])
}

// ReversedHexToIPv4 decodes the given hex representation of a reversed
// ip address to an IP object.
func ReversedHexToIPv4(h string) (net.IP, error) {
	ipBytes, err := hex.DecodeString(h)
	if err != nil {
		return nil, err
	}
	if len(ipBytes) != 4 {
		return nil, fmt.Errorf("%s is not a valid ip hex string", h)
	}
	rev := RevBytes(ipBytes)
	return net.IPv4(rev[0], rev[1], rev[2], rev[3]), nil
}

// IPv4ToReverseHex reverses the given IP address string and encodes it to
// hexadecimal.
func IPv4ToReverseHex(ip net.IP) string {
	return Pack32BinaryIPv4(RevIPv4(ip))
}

// Pack32BinaryIPv4 will pack the given IP address string to 32-bit hexadecimal format.
func Pack32BinaryIPv4(ip net.IP) string {
	ipv4Decimal := IPv4ToInt(ip)

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, uint32(ipv4Decimal))

	if err != nil {
		fmt.Println("Unable to write to buffer:", err)
	}

	// present in hexadecimal format
	result := fmt.Sprintf("%x", buf.Bytes())
	return strings.ToUpper(result)
}

// IPv4ToInt produces the decimal representation of the given IP address.
func IPv4ToInt(ip net.IP) int64 {
	ipv4Int := big.NewInt(0)
	ipv4Int.SetBytes(ip.To4())
	return ipv4Int.Int64()
}

// GetPIDByName returns the first process ID that matches the given name.
func GetPIDByName(name string) (int, error) {
	procs, err := ps.Processes()
	if err != nil {
		return 0, err
	}
	for _, proc := range procs {
		if proc.Executable() == name {
			return proc.Pid(), nil
		}
	}
	return 0, fmt.Errorf("No process found for %s", name)
}

// GetNonLoopbackAddresses returns a list of the non-loopback IP addresses
// configured on the local machine.
func GetNonLoopbackAddresses() ([]net.IP, error) {
	possibleAddrs := make([]net.IP, 0)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	for _, a := range addrs {
		var ip net.IP
		switch v := a.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if !ip.IsLoopback() {
			possibleAddrs = append(possibleAddrs, ip)
		}
	}
	return possibleAddrs, nil
}

// GetExternalAddressForProcess attempts to find the external address that a process
// is listening on.
func GetExternalAddressForProcess(name string) (net.IP, error) {
	possibleAddrs, err := GetNonLoopbackAddresses()
	if err != nil {
		return nil, err
	}
	log.Debug("Possible external addresses:", possibleAddrs)
	pid, err := GetPIDByName(name)
	if err != nil {
		return nil, err
	}
	p := fmt.Sprintf("/proc/%d/net/tcp", pid)
	log.Debugf("Scanning %q for remote listener\n", p)
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip first line
	for scanner.Scan() {
		text := scanner.Text()
		fields := strings.Fields(text)
		if len(fields) < 3 {
			return nil, errors.New("Unexpected error reading proc file, line has less than 3 fields")
		}
		remoteAddrRaw := fields[2]
		spl := strings.Split(remoteAddrRaw, ":")
		if len(spl) < 2 {
			return nil, errors.New("Unexpected error reading proc file, addr doesn't have two parts")
		}
		addrHex := spl[0]
		if procLocalhost == addrHex {
			continue
		}
		if isPossibleAddr(possibleAddrs, addrHex) {
			log.Debugf("%s appears to be listening on addr hex %s\n", name, addrHex)
			return ReversedHexToIPv4(addrHex)
		}
	}
	return nil, fmt.Errorf("Could not find an external address for %s", name)
}

var procLocalhost = Pack32BinaryIPv4(net.IPv4(1, 0, 0, 127))

func isPossibleAddr(possible []net.IP, addrHex string) bool {
	for _, p := range possible {
		if ip4 := p.To4(); ip4 != nil {
			if IPv4ToReverseHex(ip4) == addrHex {
				return true
			}
		}
	}
	return false
}
