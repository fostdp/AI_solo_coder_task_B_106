package db

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/ClickHouse/clickhouse-go/v2"
	"go.uber.org/zap"
	"stone-relic-monitor/internal/config"
	"sync"
	"time"
)

type ClickHouse struct {
	cfg         *config.Config
	DB          *sql.DB
	batchBuffer chan *batchItem
	batchWg     sync.WaitGroup
}

type batchItem struct {
	query string
	args  []interface{}
}

const (
	batchChannelSize = 10000
	batchFlushSize   = 500
	batchFlushInterval = 5 * time.Second
	writeTimeout     = 30 * time.Second
)

func NewClickHouse(cfg *config.Config) *ClickHouse {
	return &ClickHouse{cfg: cfg}
}

func (ch *ClickHouse) Connect() error {
	dsn := fmt.Sprintf(
		"clickhouse://%s:%s@%s:%d/%s?dial_timeout=10s&read_timeout=20s",
		ch.cfg.ClickHouse.Username,
		ch.cfg.ClickHouse.Password,
		ch.cfg.ClickHouse.Host,
		ch.cfg.ClickHouse.Port,
		ch.cfg.ClickHouse.Database,
	)

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("open clickhouse failed: %w", err)
	}

	db.SetMaxOpenConns(ch.cfg.ClickHouse.MaxOpenConns)
	db.SetMaxIdleConns(ch.cfg.ClickHouse.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(ch.cfg.ClickHouse.ConnMaxLifetime) * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping clickhouse failed: %w", err)
	}

	ch.DB = db

	ch.batchBuffer = make(chan *batchItem, batchChannelSize)
	ch.batchWg.Add(1)
	go ch.batchWriterLoop()

	zap.L().Info("ClickHouse connected successfully")
	return nil
}

func (ch *ClickHouse) Close() error {
	close(ch.batchBuffer)
	ch.batchWg.Wait()
	if ch.DB != nil {
		return ch.DB.Close()
	}
	return nil
}

func (ch *ClickHouse) AsyncInsert(query string, args ...interface{}) bool {
	item := &batchItem{query: query, args: args}
	select {
	case ch.batchBuffer <- item:
		return true
	default:
		zap.L().Warn("Batch buffer full, async insert dropped",
			zap.Int("buffer_len", len(ch.batchBuffer)))
		return false
	}
}

func (ch *ClickHouse) BatchInsertSync(ctx context.Context, query string, rows [][]interface{}) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	tx, err := ch.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx failed: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("prepare failed: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, args := range rows {
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			zap.L().Warn("Batch insert row failed", zap.Error(err))
			continue
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit failed: %w", err)
	}
	return inserted, nil
}

func (ch *ClickHouse) batchWriterLoop() {
	defer ch.batchWg.Done()

	var buffer []*batchItem
	currentQuery := ""
	ticker := time.NewTicker(batchFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case item, ok := <-ch.batchBuffer:
			if !ok {
				if len(buffer) > 0 {
					ch.flushBuffer(currentQuery, buffer)
				}
				return
			}
			if currentQuery == "" {
				currentQuery = item.query
			}
			if item.query != currentQuery {
				if len(buffer) > 0 {
					ch.flushBuffer(currentQuery, buffer)
				}
				currentQuery = item.query
				buffer = buffer[:0]
			}
			buffer = append(buffer, item)
			if len(buffer) >= batchFlushSize {
				ch.flushBuffer(currentQuery, buffer)
				buffer = buffer[:0]
				currentQuery = ""
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				ch.flushBuffer(currentQuery, buffer)
				buffer = buffer[:0]
				currentQuery = ""
			}
		}
	}
}

func (ch *ClickHouse) flushBuffer(query string, items []*batchItem) {
	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()

	tx, err := ch.DB.BeginTx(ctx, nil)
	if err != nil {
		zap.L().Error("Flush buffer begin tx failed", zap.Error(err), zap.Int("rows", len(items)))
		return
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		zap.L().Error("Flush buffer prepare failed", zap.Error(err))
		return
	}
	defer stmt.Close()

	flushed := 0
	for _, item := range items {
		if _, err := stmt.ExecContext(ctx, item.args...); err != nil {
			zap.L().Warn("Flush buffer row failed", zap.Error(err))
			continue
		}
		flushed++
	}

	if err := tx.Commit(); err != nil {
		zap.L().Error("Flush buffer commit failed", zap.Error(err), zap.Int("attempted", flushed))
		return
	}
	zap.L().Debug("Flushed batch buffer", zap.Int("rows", flushed))
}

func (ch *ClickHouse) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := ch.DB.ExecContext(ctx, query, args...)
	return err
}

func (ch *ClickHouse) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return ch.DB.QueryContext(ctx, query, args...)
}

func (ch *ClickHouse) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return ch.DB.QueryRowContext(ctx, query, args...)
}
