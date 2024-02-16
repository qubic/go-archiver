package store

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var ErrNotFound = errors.New("store resource not found")

type PebbleStore struct {
	db     *pebble.DB
	logger *zap.Logger
}

func NewPebbleStore(db *pebble.DB, logger *zap.Logger) *PebbleStore {
	return &PebbleStore{db: db, logger: logger}
}

func (s *PebbleStore) GetTickData(ctx context.Context, tickNumber uint64) (*protobuff.TickData, error) {
	key := tickDataKey(tickNumber)
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting tick data")
	}
	defer closer.Close()

	var td protobuff.TickData
	if err := proto.Unmarshal(value, &td); err != nil {
		return nil, errors.Wrap(err, "unmarshalling tick data to protobuff type")
	}

	return &td, err
}

func (s *PebbleStore) SetTickData(ctx context.Context, tickNumber uint64, td *protobuff.TickData) error {
	key := tickDataKey(tickNumber)
	serialized, err := proto.Marshal(td)
	if err != nil {
		return errors.Wrap(err, "serializing td proto")
	}

	err = s.db.Set(key, serialized, &pebble.WriteOptions{Sync: true})
	if err != nil {
		return errors.Wrap(err, "setting tick data")
	}

	return nil
}

func (s *PebbleStore) GetQuorumTickData(ctx context.Context, tickNumber uint64) (*protobuff.QuorumTickData, error) {
	key := quorumTickDataKey(tickNumber)
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting quorum tick data")
	}
	defer closer.Close()

	var qtd protobuff.QuorumTickData
	if err := proto.Unmarshal(value, &qtd); err != nil {
		return nil, errors.Wrap(err, "unmarshalling quorum tick data to protobuf type")
	}

	return &qtd, err
}

func (s *PebbleStore) SetQuorumTickData(ctx context.Context, tickNumber uint64, qtd *protobuff.QuorumTickData) error {
	key := quorumTickDataKey(tickNumber)
	serialized, err := proto.Marshal(qtd)
	if err != nil {
		return errors.Wrap(err, "serializing qtd proto")
	}

	err = s.db.Set(key, serialized, &pebble.WriteOptions{Sync: true})
	if err != nil {
		return errors.Wrap(err, "setting quorum tick data")
	}

	return nil
}

func (s *PebbleStore) GetComputors(ctx context.Context, epoch uint64) (*protobuff.Computors, error) {
	key := computorsKey(epoch)

	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting quorum tick data")
	}
	defer closer.Close()

	var computors protobuff.Computors
	if err := proto.Unmarshal(value, &computors); err != nil {
		return nil, errors.Wrap(err, "unmarshalling computors to protobuff type")
	}

	return &computors, nil
}

func (s *PebbleStore) SetComputors(ctx context.Context, epoch uint64, computors *protobuff.Computors) error {
	key := computorsKey(epoch)

	serialized, err := proto.Marshal(computors)
	if err != nil {
		return errors.Wrap(err, "serializing computors proto")
	}

	err = s.db.Set(key, serialized, &pebble.WriteOptions{Sync: true})
	if err != nil {
		return errors.Wrap(err, "setting computors")
	}

	return nil
}

func (s *PebbleStore) SetTickTransactions(ctx context.Context, txs *protobuff.Transactions) error {
	batch := s.db.NewBatchWithSize(len(txs.GetTransactions()))
	defer batch.Close()

	for _, tx := range txs.GetTransactions() {
		digest, err := hex.DecodeString(tx.DigestHex)
		if err != nil {
			return errors.Wrapf(err, "decoding hex digest: %s", tx.DigestHex)
		}

		key, err := tickTxKey(digest)
		if err != nil {
			return errors.Wrapf(err, "creating tx key for digest: %s", digest)
		}

		serialized, err := protojson.MarshalOptions{EmitDefaultValues: true}.Marshal(tx)
		if err != nil {
			return errors.Wrap(err, "serializing tx proto")
		}

		err = batch.Set(key, serialized, nil)
		if err != nil {
			return errors.Wrap(err, "getting tick data")
		}
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return errors.Wrap(err, "committing batch")
	}

	return nil
}

func (s *PebbleStore) GetTickTransactions(ctx context.Context, tickNumber uint64) (*protobuff.Transactions, error) {
	td, err := s.GetTickData(ctx, tickNumber)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting tick data")
	}

	txs := make([]*protobuff.Transaction, 0, len(td.TransactionDigestsHex))
	for _, digestHex := range td.TransactionDigestsHex {
		digest, err := hex.DecodeString(digestHex)
		if err != nil {
			return nil, errors.Wrapf(err, "decoding hex digest: %s", digestHex)
		}

		tx, err := s.GetTransaction(ctx, digest)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNotFound
			}

			return nil, errors.Wrapf(err, "getting tx for digest: %s", digestHex)
		}

		txs = append(txs, tx)
	}

	return &protobuff.Transactions{Transactions: txs}, nil
}

func (s *PebbleStore) GetTransaction(ctx context.Context, digest []byte) (*protobuff.Transaction, error) {
	key, err := tickTxKey(digest)
	if err != nil {
		return nil, errors.Wrap(err, "getting tx key")
	}

	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting tx")
	}
	defer closer.Close()

	var tx protobuff.Transaction
	if err := protojson.Unmarshal(value, &tx); err != nil {
		return nil, errors.Wrap(err, "unmarshalling tx to protobuff type")
	}

	return &tx, nil
}

func (s *PebbleStore) SetLastProcessedTick(ctx context.Context, tickNumber uint64) error {
	key := lastProcessedTickKey()
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, tickNumber)

	err := s.db.Set(key, value, &pebble.WriteOptions{Sync: true})
	if err != nil {
		return errors.Wrap(err, "setting last processed tick")
	}

	return nil
}

func (s *PebbleStore) GetLastProcessedTick(ctx context.Context) (uint64, error) {
	key := lastProcessedTickKey()
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return 0, ErrNotFound
		}

		return 0, errors.Wrap(err, "getting last processed tick")
	}
	defer closer.Close()

	return binary.LittleEndian.Uint64(value), nil
}


