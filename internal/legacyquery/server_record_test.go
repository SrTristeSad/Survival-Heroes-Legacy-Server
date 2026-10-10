package legacyquery

import "testing"

func TestServerRecordRoundTrip(t *testing.T) {
	want := ServerRecord{
		ID:       "legacy-1",
		District: "district_key",
		Address:  "127.0.0.1",
		Port:     2001,
		Type:     ServerTypeRecommended,
		Open:     1,
		Load:     ServerLoadGood,
		Priority: 7,
	}
	line, err := want.MarshalLine()
	if err != nil {
		t.Fatal(err)
	}
	if line != "legacy-1/district_key/127.0.0.1/2001/1/1/0/7" {
		t.Fatalf("line=%q", line)
	}
	got, err := ParseServerRecordLine(line)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestServerRecordRejectsDelimiter(t *testing.T) {
	_, err := (ServerRecord{
		ID:       "bad/name",
		Address:  "127.0.0.1",
		Port:     2001,
		Open:     1,
		Priority: 1,
	}).MarshalLine()
	if err == nil {
		t.Fatal("expected slash delimiter to be rejected")
	}
}

func TestServerRecordRequiresEightFields(t *testing.T) {
	if _, err := ParseServerRecordLine("a/b/c/1/2/3/4"); err == nil {
		t.Fatal("expected short record to be rejected")
	}
}

func TestServerRecordRejectsInvalidState(t *testing.T) {
	_, err := (ServerRecord{
		ID:       "legacy-1",
		District: "district_key",
		Address:  "127.0.0.1",
		Port:     2001,
		Type:     ServerTypeNormal,
		Open:     2,
		Load:     ServerLoadGood,
		Priority: 1,
	}).MarshalLine()
	if err == nil {
		t.Fatal("expected invalid open state to be rejected")
	}
}
