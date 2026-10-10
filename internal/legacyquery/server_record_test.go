package legacyquery

import "testing"

func TestServerRecordRoundTrip(t *testing.T) {
	want := ServerRecord{
		Text0: "legacy-1",
		Text1: "district_key",
		Text2: "127.0.0.1",
		Int3:  2001,
		Int4:  1,
		Int5:  2,
		Int6:  10000,
		Int7:  7,
	}
	line, err := want.MarshalLine()
	if err != nil {
		t.Fatal(err)
	}
	if line != "legacy-1/district_key/127.0.0.1/2001/1/2/10000/7" {
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
	_, err := (ServerRecord{Text0: "bad/name"}).MarshalLine()
	if err == nil {
		t.Fatal("expected slash delimiter to be rejected")
	}
}

func TestServerRecordRequiresEightFields(t *testing.T) {
	if _, err := ParseServerRecordLine("a/b/c/1/2/3/4"); err == nil {
		t.Fatal("expected short record to be rejected")
	}
}
