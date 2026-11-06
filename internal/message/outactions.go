package message

import (
	"encoding/base64"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

const (
	OpJettonTransfer uint32 = 0x0f8a7ea5
	OpNFTTransfer    uint32 = 0x5fcc3d14

	actionSendMsg uint32 = 0x0ec3c86d
)

type OutMsg struct {
	Op           uint32
	ValueNano    int64
	Dest         *address.Address
	ResponseDest *address.Address
}

func ParseOutMsgs(bocBase64 string) (msgs []OutMsg, err error) {
	defer func() {
		if r := recover(); r != nil {
			msgs = nil
			err = ErrMalformed
		}
	}()

	raw, derr := base64.StdEncoding.DecodeString(bocBase64)
	if derr != nil {
		return nil, ErrMalformed
	}
	root, derr := cell.FromBOC(raw)
	if derr != nil {
		return nil, ErrMalformed
	}
	sl, derr := root.BeginParse()
	if derr != nil {
		return nil, ErrMalformed
	}

	op, derr := sl.LoadUInt(32)
	if derr != nil {
		return nil, ErrMalformed
	}
	if uint32(op) != authSignedInternal {
		return nil, ErrUnsupportedOp
	}
	for i := 0; i < 3; i++ { // wallet_id, valid_until, seqno
		if _, derr = sl.LoadUInt(32); derr != nil {
			return nil, ErrMalformed
		}
	}

	listRef, derr := sl.LoadMaybeRef() // out_actions: (1 bit) ^OutList
	if derr != nil {
		return nil, ErrMalformed
	}
	if listRef == nil {
		return nil, nil
	}

	cur := listRef
	for cur.RefsNum() > 0 {
		prev, rerr := cur.LoadRef()
		if rerr != nil {
			return nil, ErrMalformed
		}
		aop, aerr := cur.LoadUInt(32)
		if aerr != nil {
			return nil, ErrMalformed
		}
		if uint32(aop) != actionSendMsg {
			break
		}
		if _, aerr = cur.LoadUInt(8); aerr != nil { // mode
			return nil, ErrMalformed
		}
		msgRef, merr := cur.LoadRef()
		if merr != nil {
			return nil, ErrMalformed
		}

		var mr tlb.MessageRelaxed
		if lerr := mr.LoadFromCell(msgRef); lerr == nil && mr.Info.DstAddr != nil {
			om := OutMsg{Dest: mr.Info.DstAddr}
			if mr.Info.Amount != nil {
				om.ValueNano = mr.Info.Amount.Int64()
			}
			if mr.Body.InRef && mr.Body.Ref != nil {
				parseTransferBody(mr.Body.Ref, &om)
			}
			msgs = append(msgs, om)
		}

		cur = prev
	}

	return msgs, nil
}

func parseTransferBody(bodyCell *cell.Cell, om *OutMsg) {
	b, err := bodyCell.BeginParse()
	if err != nil {
		return
	}
	if b.BitsLeft() < 32 {
		return
	}
	op, err := b.LoadUInt(32)
	if err != nil {
		return
	}
	om.Op = uint32(op)

	switch uint32(op) {
	case OpJettonTransfer:
		if _, err = b.LoadUInt(64); err != nil { // query_id
			return
		}
		if _, err = b.LoadBigCoins(); err != nil { // amount
			return
		}
		if _, err = b.LoadAddr(); err != nil { // destination
			return
		}
		if resp, rerr := b.LoadAddr(); rerr == nil { // response_destination
			om.ResponseDest = resp
		}
	case OpNFTTransfer:
		if _, err = b.LoadUInt(64); err != nil { // query_id
			return
		}
		if _, err = b.LoadAddr(); err != nil { // new_owner
			return
		}
		if resp, rerr := b.LoadAddr(); rerr == nil { // response_destination
			om.ResponseDest = resp
		}
	}
}

func ForwardEligible(msgs []OutMsg, relayer *address.Address, capNano int64) []OutMsg {
	if relayer == nil {
		return nil
	}
	var out []OutMsg
	for _, m := range msgs {
		if m.Op != OpJettonTransfer && m.Op != OpNFTTransfer {
			continue
		}
		if m.ResponseDest == nil || !m.ResponseDest.Equals(relayer) {
			continue
		}
		if m.ValueNano <= 0 || m.ValueNano > capNano {
			continue
		}
		out = append(out, m)
	}
	return out
}
