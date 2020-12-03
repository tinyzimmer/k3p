package cluster

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
	"strconv"
	"strings"

	ps "github.com/mitchellh/go-ps"
	"github.com/tinyzimmer/k3p/pkg/log"
)

var procK3sPort = strings.ToUpper(strconv.FormatInt(6443, 16))
var procLocalhost = strings.ToUpper(pack32BinaryIP4("1.0.0.127"))

func getExternalK3sAddr() (addr string, err error) {
	procs, err := ps.Processes()
	if err != nil {
		return "", err
	}
	for _, proc := range procs {
		if proc.Executable() != "k3s-server" {
			continue
		}
		p := fmt.Sprintf("/proc/%d/net/tcp", proc.Pid())
		log.Debugf("Scanning %q for remote port\n", p)
		f, err := os.Open(p)
		if err != nil {
			return "", err
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		scanner.Scan() // skip first line
		for scanner.Scan() {
			text := scanner.Text()
			fields := strings.Fields(text)
			if len(fields) < 3 {
				return "", errors.New("Unexpected error reading proc file, line has less than 3 fields")
			}
			remoteAddrRaw := fields[2]
			spl := strings.Split(remoteAddrRaw, ":")
			if len(spl) < 2 {
				return "", errors.New("Unexpected error reading proc file, addr doesn't have two parts")
			}
			addrHex, portHex := spl[0], spl[1]
			if procLocalhost == addrHex {
				continue
			}
			if portHex == procK3sPort {
				return hexToIP(addrHex)
			}

		}
	}
	return "", errors.New("Could not determine k3s external address")
}

func pack32BinaryIP4(ip4Address string) string {
	ipv4Decimal := ip4toInt(net.ParseIP(ip4Address))

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, uint32(ipv4Decimal))

	if err != nil {
		fmt.Println("Unable to write to buffer:", err)
	}

	// present in hexadecimal format
	result := fmt.Sprintf("%x", buf.Bytes())
	return result
}

func ip4toInt(IPv4Address net.IP) int64 {
	ipv4Int := big.NewInt(0)
	ipv4Int.SetBytes(IPv4Address.To4())
	return ipv4Int.Int64()
}

func hexToIP(h string) (string, error) {
	ipBytes, err := hex.DecodeString(h)
	if err != nil {
		return "", err
	}
	if len(ipBytes) != 4 {
		return "", fmt.Errorf("%s is not a valid ip hex string", h)
	}
	return net.IP(rev(ipBytes)).String(), nil
}

func rev(b []byte) []byte {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return b
}
