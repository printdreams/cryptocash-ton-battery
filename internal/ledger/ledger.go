package ledger

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OpType string

const (
	OpCredit  OpType = "credit"
	OpReserve OpType = "reserve"
	OpSettle  OpType = "settle"
	OpRelease OpType = "release"
)

const accountsCollection = "accounts"

var (
	ErrInvalidAmount        = errors.New("invalid-amount")
	ErrInsufficientBalance  = errors.New("insufficient-charges")
	ErrInsufficientReserved = errors.New("insufficient-reserved")
	ErrMissingIdempotency   = errors.New("missing-idempotency-key")
	ErrUnknownOp            = errors.New("unknown-op")
)

type Ref struct {
	IdempotencyKey string
	Type           string
	Reason         string
}

type Account struct {
	UserID   string
	Balance  int64
	Reserved int64
}

func (a Account) Available() int64 {
	return a.Balance - a.Reserved
}

type Entry struct {
	ID             string    `firestore:"-"`
	Op             string    `firestore:"op"`
	Amount         int64     `firestore:"amount"`
	BalanceAfter   int64     `firestore:"balance_after"`
	ReservedAfter  int64     `firestore:"reserved_after"`
	IdempotencyKey string    `firestore:"idempotency_key"`
	RefType        string    `firestore:"ref_type"`
	Reason         string    `firestore:"reason"`
	CreatedAt      time.Time `firestore:"created_at"`
}

type Result struct {
	Account    Account
	Entry      Entry
	Idempotent bool
}

func applyOp(balance, reserved, amount int64, op OpType) (int64, int64, error) {
	if amount <= 0 {
		return 0, 0, ErrInvalidAmount
	}
	switch op {
	case OpCredit:
		return balance + amount, reserved, nil
	case OpReserve:
		if balance-reserved < amount {
			return 0, 0, ErrInsufficientBalance
		}
		return balance, reserved + amount, nil
	case OpSettle:
		if reserved < amount {
			return 0, 0, ErrInsufficientReserved
		}
		return balance - amount, reserved - amount, nil
	case OpRelease:
		if reserved < amount {
			return 0, 0, ErrInsufficientReserved
		}
		return balance, reserved - amount, nil
	default:
		return 0, 0, ErrUnknownOp
	}
}

type Store struct {
	fs *firestore.Client
}

func NewStore(fs *firestore.Client) *Store {
	return &Store{fs: fs}
}

func (s *Store) Credit(ctx context.Context, userID string, amount int64, ref Ref) (*Result, error) {
	return s.apply(ctx, userID, amount, OpCredit, ref)
}

func (s *Store) Reserve(ctx context.Context, userID string, amount int64, ref Ref) (*Result, error) {
	return s.apply(ctx, userID, amount, OpReserve, ref)
}

func (s *Store) Settle(ctx context.Context, userID string, amount int64, ref Ref) (*Result, error) {
	return s.apply(ctx, userID, amount, OpSettle, ref)
}

func (s *Store) Release(ctx context.Context, userID string, amount int64, ref Ref) (*Result, error) {
	return s.apply(ctx, userID, amount, OpRelease, ref)
}

func (s *Store) Balance(ctx context.Context, userID string) (Account, error) {
	acc := Account{UserID: userID}
	snap, err := s.fs.Collection(accountsCollection).Doc(userID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return acc, nil
		}
		return acc, err
	}
	acc.Balance, acc.Reserved = readBalances(snap)
	return acc, nil
}

func (s *Store) History(ctx context.Context, userID string, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 50
	}
	iter := s.fs.Collection(accountsCollection).Doc(userID).Collection("entries").
		OrderBy("created_at", firestore.Desc).Limit(limit).Documents(ctx)
	defer iter.Stop()

	entries := make([]Entry, 0, limit)
	for {
		doc, err := iter.Next()
		if err != nil {
			if status.Code(err) == codes.NotFound {
				break
			}
			if err.Error() == "no more items in iterator" {
				break
			}
			return nil, err
		}
		var e Entry
		if err := doc.DataTo(&e); err != nil {
			return nil, err
		}
		e.ID = doc.Ref.ID
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) apply(ctx context.Context, userID string, amount int64, op OpType, ref Ref) (*Result, error) {
	if ref.IdempotencyKey == "" {
		return nil, ErrMissingIdempotency
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	accRef := s.fs.Collection(accountsCollection).Doc(userID)
	idemRef := accRef.Collection("idem").Doc(ref.IdempotencyKey)
	entriesCol := accRef.Collection("entries")

	result := &Result{}

	err := s.fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		idemSnap, err := tx.Get(idemRef)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if err == nil && idemSnap.Exists() {
			var e Entry
			if derr := idemSnap.DataTo(&e); derr != nil {
				return derr
			}
			e.ID = idemSnap.Ref.ID
			result.Entry = e
			result.Account = Account{UserID: userID, Balance: e.BalanceAfter, Reserved: e.ReservedAfter}
			result.Idempotent = true
			return nil
		}

		accSnap, err := tx.Get(accRef)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		var balance, reserved int64
		if err == nil && accSnap.Exists() {
			balance, reserved = readBalances(accSnap)
		}

		newBalance, newReserved, aerr := applyOp(balance, reserved, amount, op)
		if aerr != nil {
			return aerr
		}

		now := time.Now().UTC()
		entryID := uuid.NewString()
		entry := Entry{
			ID:             entryID,
			Op:             string(op),
			Amount:         amount,
			BalanceAfter:   newBalance,
			ReservedAfter:  newReserved,
			IdempotencyKey: ref.IdempotencyKey,
			RefType:        ref.Type,
			Reason:         ref.Reason,
			CreatedAt:      now,
		}

		if err := tx.Set(accRef, map[string]interface{}{
			"balance":    newBalance,
			"reserved":   newReserved,
			"updated_at": now,
		}, firestore.MergeAll); err != nil {
			return err
		}
		if err := tx.Create(entriesCol.Doc(entryID), entry); err != nil {
			return err
		}
		if err := tx.Create(idemRef, entry); err != nil {
			return err
		}

		result.Entry = entry
		result.Account = Account{UserID: userID, Balance: newBalance, Reserved: newReserved}
		result.Idempotent = false
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

type Report struct {
	UserID           string
	AccountBalance   int64
	AccountReserved  int64
	ComputedBalance  int64
	ComputedReserved int64
	Consistent       bool
}

func deltaFor(op OpType, amount int64) (int64, int64) {
	switch op {
	case OpCredit:
		return amount, 0
	case OpReserve:
		return 0, amount
	case OpSettle:
		return -amount, -amount
	case OpRelease:
		return 0, -amount
	}
	return 0, 0
}

func (s *Store) Reconcile(ctx context.Context, userID string) (*Report, error) {
	acc, err := s.Balance(ctx, userID)
	if err != nil {
		return nil, err
	}

	var computedBalance, computedReserved int64
	iter := s.fs.Collection(accountsCollection).Doc(userID).Collection("entries").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var e Entry
		if err := doc.DataTo(&e); err != nil {
			return nil, err
		}
		db, dr := deltaFor(OpType(e.Op), e.Amount)
		computedBalance += db
		computedReserved += dr
	}

	return &Report{
		UserID:           userID,
		AccountBalance:   acc.Balance,
		AccountReserved:  acc.Reserved,
		ComputedBalance:  computedBalance,
		ComputedReserved: computedReserved,
		Consistent:       computedBalance == acc.Balance && computedReserved == acc.Reserved,
	}, nil
}

func readBalances(snap *firestore.DocumentSnapshot) (int64, int64) {
	var balance, reserved int64
	if v, err := snap.DataAt("balance"); err == nil {
		if n, ok := v.(int64); ok {
			balance = n
		}
	}
	if v, err := snap.DataAt("reserved"); err == nil {
		if n, ok := v.(int64); ok {
			reserved = n
		}
	}
	return balance, reserved
}
