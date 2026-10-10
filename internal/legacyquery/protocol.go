package legacyquery

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	RequestSize     = 28
	RequestMagic0   = 24
	RequestCommand  = 1001
	RequestVersion0 = 1
	RequestVersion1 = 1
	RequestMagic1   = 22222
)

// Request is the 28-byte query sent by Survival Heroes 3.3.3 to the
// [server_query] endpoint from assets/config.ini.
//
// The client serializer CMsgBuf::SetInt stores native uint32 values directly.
// The released Android client is little-endian ARM, therefore the wire request
// is seven consecutive little-endian uint32 values.
type Request struct {
	Header0 uint32
	Command uint32
	Ver0    uint32
	Ver1    uint32
	Header1 uint32
	Issuer  uint32
	GameID  uint32
}

func ParseRequest(data []byte) (Request, error) {
	if len(data) != RequestSize {
		return Request{}, fmt.Errorf("legacy query: expected %d request bytes, got %d", RequestSize, len(data))
	}
	return Request{
		Header0: binary.LittleEndian.Uint32(data[0:4]),
		Command: binary.LittleEndian.Uint32(data[4:8]),
		Ver0:    binary.LittleEndian.Uint32(data[8:12]),
		Ver1:    binary.LittleEndian.Uint32(data[12:16]),
		Header1: binary.LittleEndian.Uint32(data[16:20]),
		Issuer:  binary.LittleEndian.Uint32(data[20:24]),
		GameID:  binary.LittleEndian.Uint32(data[24:28]),
	}, nil
}

func (r Request) IsKnownShape() bool {
	return r.Header0 == RequestMagic0 &&
		r.Command == RequestCommand &&
		r.Ver0 == RequestVersion0 &&
		r.Ver1 == RequestVersion1 &&
		r.Header1 == RequestMagic1
}

func (r Request) MarshalBinary() []byte {
	out := make([]byte, RequestSize)
	values := [...]uint32{r.Header0, r.Command, r.Ver0, r.Ver1, r.Header1, r.Issuer, r.GameID}
	for i, v := range values {
		binary.LittleEndian.PutUint32(out[i*4:(i+1)*4], v)
	}
	return out
}

// ResponseBody is what CPackageHeadSocket::GetRecvInfo feeds into CMsgBuf.
// The client consumes five uint32 values and then Count NUL-terminated strings.
// The first four integers are currently ignored by the 3.3.3 client, but we
// preserve the request-family constants instead of relying on that accident.
type ResponseBody struct {
	Header0 uint32
	Command uint32
	Ver0    uint32
	Ver1    uint32
	Servers []string
}

func DefaultResponseBody(servers []string) ResponseBody {
	return ResponseBody{
		Header0: RequestMagic0,
		Command: RequestCommand,
		Ver0:    RequestVersion0,
		Ver1:    RequestVersion1,
		Servers: servers,
	}
}

func (b ResponseBody) MarshalBinary() ([]byte, error) {
	for _, s := range b.Servers {
		for i := 0; i < len(s); i++ {
			if s[i] == 0 {
				return nil, errors.New("legacy query: server record contains NUL")
			}
		}
	}

	n := 5 * 4
	for _, s := range b.Servers {
		n += len(s) + 1
	}
	out := make([]byte, n)
	values := [...]uint32{b.Header0, b.Command, b.Ver0, b.Ver1, uint32(len(b.Servers))}
	for i, v := range values {
		binary.LittleEndian.PutUint32(out[i*4:(i+1)*4], v)
	}
	pos := len(values) * 4
	for _, s := range b.Servers {
		copy(out[pos:], s)
		pos += len(s)
		out[pos] = 0
		pos++
	}
	return out, nil
}

// MarshalUncompressedEnvelope reproduces the framing observed in
// CPackageHeadSocket::GetRecvInfo:
//
//   uint32 remaining_length  // bytes after this field
//   uint32 unpacked_size     // zero means payload is stored/raw
//   byte[] payload
//
// GetRecvInfo waits for remaining_length+4 total bytes. For an uncompressed
// response remaining_length is 4+len(payload).
func MarshalUncompressedEnvelope(body []byte) []byte {
	out := make([]byte, 8+len(body))
	binary.LittleEndian.PutUint32(out[0:4], uint32(4+len(body)))
	binary.LittleEndian.PutUint32(out[4:8], 0)
	copy(out[8:], body)
	return out
}

func BuildResponse(servers []string) ([]byte, error) {
	body, err := DefaultResponseBody(servers).MarshalBinary()
	if err != nil {
		return nil, err
	}
	return MarshalUncompressedEnvelope(body), nil
}
