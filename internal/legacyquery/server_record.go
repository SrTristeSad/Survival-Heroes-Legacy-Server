package legacyquery

import (
	"fmt"
	"strconv"
	"strings"
)

// ServerRecord is the exact textual record shape consumed by the archived
// Survival Heroes 3.3.3 FormLoginServers code.
//
// Static analysis confirms eight slash-delimited fields in this order:
// three strings followed by five decimal integers. The semantic names of the
// final fields are intentionally not guessed here; keep them positional until
// each use is independently confirmed in the client.
type ServerRecord struct {
	Text0 string
	Text1 string
	Text2 string
	Int3  int32
	Int4  int32
	Int5  int32
	Int6  int32
	Int7  int32
}

func validateServerText(v string) error {
	if strings.ContainsAny(v, "/\x00\r\n") {
		return fmt.Errorf("legacy query: server text field contains forbidden delimiter/control character")
	}
	return nil
}

// MarshalLine encodes the exact line grammar written by the historical query
// service into server_list.ini. WriteServerListFile adds CRLF separately.
func (r ServerRecord) MarshalLine() (string, error) {
	for _, v := range [...]string{r.Text0, r.Text1, r.Text2} {
		if err := validateServerText(v); err != nil {
			return "", err
		}
	}
	return strings.Join([]string{
		r.Text0,
		r.Text1,
		r.Text2,
		strconv.FormatInt(int64(r.Int3), 10),
		strconv.FormatInt(int64(r.Int4), 10),
		strconv.FormatInt(int64(r.Int5), 10),
		strconv.FormatInt(int64(r.Int6), 10),
		strconv.FormatInt(int64(r.Int7), 10),
	}, "/"), nil
}

// ParseServerRecordLine is primarily a regression/debug helper for recovered
// records. The 3.3.3 client requires at least eight tokens; the revival server
// deliberately emits exactly eight so its behavior stays deterministic.
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
	for i := 0; i < len(nums); i++ {
		v, err := strconv.ParseInt(parts[i+3], 10, 32)
		if err != nil {
			return ServerRecord{}, fmt.Errorf("legacy query: field %d is not int32: %w", i+3, err)
		}
		nums[i] = int32(v)
	}
	return ServerRecord{
		Text0: parts[0], Text1: parts[1], Text2: parts[2],
		Int3: nums[0], Int4: nums[1], Int5: nums[2], Int6: nums[3], Int7: nums[4],
	}, nil
}
