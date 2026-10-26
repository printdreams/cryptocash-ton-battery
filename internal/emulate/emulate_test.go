package emulate

import "testing"

func userTrace(userTx apiTransaction, grandchildren []apiTrace) apiTrace {
	return apiTrace{
		Transaction: apiTransaction{Success: true, TotalFees: 1000, Account: apiAddress{Address: "0:relayer"}},
		Children: []apiTrace{
			{Transaction: userTx, Children: grandchildren},
		},
	}
}

func TestSummarizeSuccess(t *testing.T) {
	root := userTrace(apiTransaction{
		Success:   true,
		TotalFees: 500,
		Account:   apiAddress{Address: "0:user"},
		OutMsgs: []apiMessage{
			{Destination: apiAddress{Address: "0:dest1"}},
			{Destination: apiAddress{Address: "0:dest2"}},
		},
	}, nil)

	res := summarize(root)
	if !res.Success {
		t.Fatalf("expected success")
	}
	if res.TotalFeesNano != 500 {
		t.Fatalf("expected user-wallet fees 500, got %d", res.TotalFeesNano)
	}
	if res.OutMessages != 2 {
		t.Fatalf("expected 2 out messages, got %d", res.OutMessages)
	}
	if len(res.Destinations) != 2 || res.Destinations[0] != "0:dest1" {
		t.Fatalf("unexpected destinations: %v", res.Destinations)
	}
}

func TestSummarizeUserTxAborted(t *testing.T) {
	root := userTrace(apiTransaction{
		Success:   false,
		Aborted:   true,
		TotalFees: 200,
		Account:   apiAddress{Address: "0:user"},
	}, nil)

	res := summarize(root)
	if res.Success {
		t.Fatalf("expected failure when the user wallet tx aborted")
	}
	if res.TotalFeesNano != 200 {
		t.Fatalf("expected user-wallet fees 200, got %d", res.TotalFeesNano)
	}
}

func TestSummarizeUninitRecipientIgnored(t *testing.T) {
	root := userTrace(apiTransaction{
		Success:   true,
		TotalFees: 450,
		Account:   apiAddress{Address: "0:user"},
		OutMsgs:   []apiMessage{{Destination: apiAddress{Address: "0:newwallet"}}},
	}, []apiTrace{
		{Transaction: apiTransaction{Success: false, Aborted: true, Account: apiAddress{Address: "0:newwallet"}}},
	})

	res := summarize(root)
	if !res.Success {
		t.Fatalf("a skipped/uninit recipient must not fail the send")
	}
	if res.OutMessages != 1 {
		t.Fatalf("expected 1 out message, got %d", res.OutMessages)
	}
}

func TestSummarizeJettonSingleOutMessage(t *testing.T) {
	root := userTrace(apiTransaction{
		Success:   true,
		TotalFees: 600,
		Account:   apiAddress{Address: "0:user"},
		OutMsgs:   []apiMessage{{Destination: apiAddress{Address: "0:userjettonwallet"}}},
	}, []apiTrace{
		{Transaction: apiTransaction{Success: true, Account: apiAddress{Address: "0:userjettonwallet"}}, Children: []apiTrace{
			{Transaction: apiTransaction{Success: true, Account: apiAddress{Address: "0:recipjettonwallet"}}, Children: []apiTrace{
				{Transaction: apiTransaction{Success: true, Account: apiAddress{Address: "0:recipient"}}},
				{Transaction: apiTransaction{Success: true, Account: apiAddress{Address: "0:user"}}},
			}},
		}},
	})

	res := summarize(root)
	if !res.Success {
		t.Fatalf("expected success for a jetton transfer")
	}
	if res.OutMessages != 1 {
		t.Fatalf("jetton transfer signs one message; got %d", res.OutMessages)
	}
}

func TestSummarizeRelayerTxFailed(t *testing.T) {
	root := apiTrace{
		Transaction: apiTransaction{Success: false, Aborted: true, TotalFees: 1000},
		Children: []apiTrace{
			{Transaction: apiTransaction{Success: true, TotalFees: 500}},
		},
	}

	res := summarize(root)
	if res.Success {
		t.Fatalf("expected failure when the relayer tx aborted")
	}
}

func TestSummarizeNoUserTx(t *testing.T) {
	root := apiTrace{
		Transaction: apiTransaction{Success: true, TotalFees: 1000},
	}

	res := summarize(root)
	if res.Success {
		t.Fatalf("expected failure when the relayer produced no message to the user wallet")
	}
}
