package legacyquery

import (
	"fmt"
	"strconv"
	"strings"
)

type ServerType int32

const (
	ServerTypeNormal      ServerType = 0
	ServerTypeRecommended ServerType = 1
	ServerTypeNew         ServerType = 2
)

type ServerLoad int32

const (
	ServerLoadGood   ServerLoad = 0
	ServerLoadNormal ServerLoad = 1
	ServerLoadHot    ServerLoad = 2
)

// ServerRecord is the exact slash-delimited record consumed by the archived
// Survival Heroes 3.3.3 FormLoginServers code.
//
// Recovered order:
//
//	server_id / district_key / address / port / type / open / load / priority
//
// type: 0=normal, 1=recommended, 2=new.
// open: 0=stopped, 1=open.
// load: 0=good, 1=normal, 2=hot.
// priority is used when choosing between recommended servers; lower wins.
// Issuer is not part of this line and is maintained separately by the login UI.
type ServerRecord struct {
	ID       string
	District string
	Address  string
	Port     int32
	Type     ServerType
	Open     int32
	Load     ServerLoad
	Priority int32
}

func validateServerText(v string) error {
	if strings.ContainsAny(v, "/\x00\r\n") {
		return fmt.Errorf("legacy query: server text field contains forbidden delimiter/control character")
	}
	return nil
}

func (r ServerRecord) Validate() error {
	for _, v := range [...]string{r.ID, r.District, r.Address} {
		if err := validateServerText(v); err != nil {
			return err
		}
	}
	if r.Port < 1 || r.Port > 65535 {
		return fmt.Errorf("legacy query: server port out of range: %d", r.Port)
	}
	if r.Type < ServerTypeNormal || r.Type > ServerTypeNew {
		return fmt.Errorf("legacy query: unsupported server type: %d", r.Type)
	}
	if r.Open != 0 && r.Open != 1 {
		return fmt.Errorf("legacy query: open must be 0 or 1, got %d", r.Open)
	}
	if r.Load < ServerLoadGood || r.Load > ServerLoadHot {
		return fmt.Errorf("legacy query: unsupported server load: %d", r.Load)
	}
	if r.Priority < 0 {
		return fmt.Errorf("legacy query: priority cannot be negative")
	}
	return nil
}

// MarshalLine encodes one historical server_list.ini line. The query client
// writes CRLF after each returned NUL-terminated string.
func (r ServerRecord) MarshalLine() (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	return strings.Join([]string{
		r.ID,
		r.District,
		r.Address,
		strconv.FormatInt(int64(r.Port), 10),
		strconv.FormatInt(int64(r.Type), 10),
		strconv.FormatInt(int64(r.Open), 10),
		strconv.FormatInt(int64(r.Load), 10),
		strconv.FormatInt(int64(r.Priority), 10),
	}, "/"), nil
}

func ParseServerRecordLine(line string) (ServerRecord, error) {
	parts := strings.Split(line, "/")
	if len(parts) != 8 {
		return ServerRecord{}, fmt.Errorf("legacy query: expected 8 server-record fields, got %d", len(parts))
	}
	for i := 0; i < 3; i++ {
		if err := validateServerText(parts[i]); err != nil {
			return ServerRecord{}, fmt.Errorf("legacy query: field %d: %w", i, err)
		}
	}
	var nums [5]int32
	for i := range nums {
		v, err := strconv.ParseInt(parts[i+3], 10, 32)
		if err != nil {
			return ServerRecord{}, fmt.Errorf("legacy query: field %d is not int32: %w", i+3, err)
		}
		nums[i] = int32(v)
	}
	r := ServerRecord{
		ID:       parts[0],
		District: parts[1],
		Address:  parts[2],
		Port:     nums[0],
		Type:     ServerType(nums[1]),
		Open:     nums[2],
		Load:     ServerLoad(nums[3]),
		Priority: nums[4],
	}
	if err := r.Validate(); err != nil {
		return ServerRecord{}, err
	}
	return r, nil
}
