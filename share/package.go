package towshare

import (
	"encoding/binary"
	"fmt"
	"net/url"
)

const (
	PackageTypeOpen  = 0x01 // open socket
	PackageTypeData  = 0x02 // socket get data
	PackageTypeClose = 0x03 // close socket
	PackageTypeError = 0x04 // error
	PackageTypeAlive = 0x05 // keep alive
	PackageTypeGetId = 0x06 // get session id
)

type Package struct {
	Type    uint8
	Payload []byte
	Remote  string
	Id      uint16
	TCP     bool
	UDP     bool   // TODO
	Error   string // error message
}

// from network
func NewPackage(b []byte) *Package {

	p := &Package{
		Type: b[0],
	}

	l := uint64(uint8(b[1]))
	offset := 2

	if l <= 125 {
		// do nothing
	} else if l == 126 {
		l = uint64(binary.BigEndian.Uint16(b[2:4]))
		offset += 2
	} else if l == 127 {
		l = uint64(binary.BigEndian.Uint64(b[2:10]))
		offset += 8
	} else {
		// throw error
	}

	if offset+int(l) == len(b) {
		p.Payload = b[offset : offset+int(l)]
	} else {
		// TODO: throw error
		fmt.Printf("Package size error: l=%d, len(b)=%d", l, len(b))
	}

	p.parsePayload()

	return p
}

func MakeKeepAlivePackage() []byte {
	return []byte{PackageTypeAlive, 0x00}
}

func MakePayloadPackageWithId(t uint8, id uint16, b []byte) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, id)
	return MakePayloadPackage(t, append(buf, b...))
}

// send data to network
func MakePayloadPackage(t uint8, b []byte) []byte {
	l := len(b)
	buf := []byte{t}
	if l <= 125 {
		buf = append(buf, uint8(l))
	} else if uint16(l) <= ^uint16(0) {
		lb := make([]byte, 2)
		binary.BigEndian.PutUint16(lb, uint16(l))
		buf = append(buf, 126)
		buf = append(buf, lb...)
	} else if uint64(l) <= ^uint64(0) {
		lb := make([]byte, 8)
		binary.BigEndian.PutUint64(lb, uint64(l))
		buf = append(buf, 127)
		buf = append(buf, lb...)
	}

	buf = append(buf, b...)

	return buf
}

func MakeErrorPackage(id uint16, s string) []byte {
	return MakePayloadPackageWithId(PackageTypeError, id, []byte(s))
}
func MakeClosePackage(id uint16) []byte {
	return MakePayloadPackageWithId(PackageTypeClose, id, nil)
}
func MakeGetIdPackage(id uint16) []byte {
	return MakePayloadPackageWithId(PackageTypeGetId, id, nil)
}
func MakeDataPackage(id uint16, b []byte) []byte {
	return MakePayloadPackageWithId(PackageTypeData, id, b)
}

func (p *Package) parsePayload() error {
	switch p.Type {
	case PackageTypeOpen:

		p.Id = binary.BigEndian.Uint16(p.Payload[0:2])
		u, err := url.Parse(string(p.Payload[2:]))
		if err != nil {
			return err
		}

		p.TCP = u.Scheme == "tcp"
		p.UDP = u.Scheme == "udp"
		p.Remote = u.Host

	case PackageTypeClose:
		p.Id = binary.BigEndian.Uint16(p.Payload)
	case PackageTypeData:
		p.Id = binary.BigEndian.Uint16(p.Payload[0:2])
		p.Payload = p.Payload[2:]
	case PackageTypeError:
		p.Id = binary.BigEndian.Uint16(p.Payload[0:2])
		p.Error = string(p.Payload[2:])
	}

	return nil
}
