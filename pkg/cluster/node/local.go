package node

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/mitchellh/go-ps"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

// Local returns a new Node pointing at the local system.
func Local() types.Node {
	return &localNode{}
}

type localNode struct{}

func (l *localNode) GetType() types.NodeType { return types.NodeLocal }

func (l *localNode) MkdirAll(dir string) error {
	log.Debugf("Ensuring local system directory %q with mode 0755\n", dir)
	return os.MkdirAll(dir, 0755)
}

func (l *localNode) Close() error { return nil }

func (l *localNode) GetFile(f string) (io.ReadCloser, error) { return os.Open(f) }

// size is ignored for local nodes
func (l *localNode) WriteFile(rdr io.ReadCloser, dest string, mode string, size int64) error {
	defer rdr.Close()
	if err := l.MkdirAll(path.Dir(dest)); err != nil {
		return err
	}
	log.Debugf("Writing file to local system at %q with mode %q\n", dest, mode)
	u, err := strconv.ParseUint(mode, 0, 16)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, os.FileMode(u))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, rdr)
	return err
}

func (l *localNode) Execute(opts *types.ExecuteOptions) error {
	cmd := buildCmdFromExecOpts(opts)
	log.Debug("Executing command on local system:", redactSecrets(cmd, opts.Secrets))
	c := exec.Command("/bin/sh", "-c", cmd)
	outPipe, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	errPipe, err := c.StderrPipe()
	if err != nil {
		return err
	}
	go log.TailReader(opts.LogPrefix, outPipe)
	go log.TailReader(opts.LogPrefix, errPipe)
	return c.Run()
}

func (l *localNode) GetK3sAddress() (string, error) {
	return getExternalK3sAddr()
}

var procLocalhost = strings.ToUpper(pack32BinaryIP4("1.0.0.127"))

func getExternalK3sAddr() (addr string, err error) {
	possibleAddrs := make([]net.IP, 0)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
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
			log.Debug("Found potential k3s IP", ip)
			possibleAddrs = append(possibleAddrs, ip)
		}
	}
	log.Debug("Possible API addresses:", possibleAddrs)
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
			addrHex := spl[0]
			if procLocalhost == addrHex {
				continue
			}
			if isPossibleAddr(possibleAddrs, addrHex) {
				log.Debug("K3s appears to be listening on addr hex", addrHex)
				return hexToIP(addrHex)
			}
		}
	}
	return "", errors.New("Could not determine k3s external address")
}

func isPossibleAddr(possible []net.IP, addrHex string) bool {
	for _, p := range possible {
		if ip4 := p.To4(); ip4 != nil {
			if toIPHex(ip4) == addrHex {
				return true
			}
		}
	}
	return false
}

func toIPHex(ip net.IP) string {
	return strings.ToUpper(pack32BinaryIP4(fmt.Sprintf("%v.%v.%v.%v", ip[3], ip[2], ip[1], ip[0])))
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
