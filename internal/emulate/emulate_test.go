package emulate

import "testing"

func TestSummarizeSuccess(t *testing.T) {
	root := apiTrace{
		Transaction: apiTransaction{Success: true, TotalFees: 1000, Account: apiAddress{Address: "0:root"}},
		Children: []apiTrace{
			{Transaction: apiTransaction{Success: true, TotalFees: 500, Account: apiAddress{Address: "0:dest1"}}},
			{Transaction: apiTransaction{Success: true, TotalFees: 300, Account: apiAddress{Address: "0:dest2"}}},
		},
	}

	res := summarize(root)
	if !res.Success {
		t.Fatalf("expected success")
	}
	if res.TotalFeesNano != 1800 {
		t.Fatalf("expected total fees 1800, got %d", res.TotalFeesNano)
	}
	if res.OutMessages != 2 {
		t.Fatalf("expected 2 out messages, got %d", res.OutMessages)
	}
	if len(res.Destinations) != 2 || res.Destinations[0] != "0:dest1" {
		t.Fatalf("unexpected destinations: %v", res.Destinations)
	}
}

func TestSummarizeAbortedChild(t *testing.T) {
	root := apiTrace{
		Transaction: apiTransaction{Success: true, TotalFees: 1000},
		Children: []apiTrace{
			{Transaction: apiTransaction{Success: false, Aborted: true, TotalFees: 200, Account: apiAddress{Address: "0:dest"}}},
		},
	}

	res := summarize(root)
	if res.Success {
		t.Fatalf("expected failure when a child aborted")
	}
	if res.TotalFeesNano != 1200 {
		t.Fatalf("expected total fees 1200, got %d", res.TotalFeesNano)
	}
}
