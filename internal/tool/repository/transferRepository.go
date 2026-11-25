package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

type TransferRepository struct{ db *pgxpool.Pool }

func NewTransferRepo(db *pgxpool.Pool) *TransferRepository { return &TransferRepository{db: db} }

func transferDBError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return tool.ErrTransferNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return tool.ErrTransferDuplicate
	}
	return err
}

func (r *TransferRepository) CreateChannel(ctx context.Context, userID int64, name string) (*toolmodel.TransferChannel, error) {
	c := &toolmodel.TransferChannel{Name: name}
	err := r.db.QueryRow(ctx, `INSERT INTO transfer_channels(user_id,name) VALUES($1,$2)
 RETURNING id,created_at,updated_at`, userID, name).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	return c, transferDBError(err)
}

func (r *TransferRepository) ListChannels(ctx context.Context, userID int64) ([]toolmodel.TransferChannel, error) {
	rows, err := r.db.Query(ctx, `SELECT c.id,c.name,c.created_at,c.updated_at,
 COUNT(i.id) FILTER(WHERE i.kind='file'), COUNT(i.id) FILTER(WHERE i.kind='text')
 FROM transfer_channels c LEFT JOIN transfer_items i ON i.channel_id=c.id AND i.state='active' AND i.created_at>NOW()-INTERVAL '360 hours'
 WHERE c.user_id=$1 AND NOT c.deleted GROUP BY c.id ORDER BY c.created_at,c.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]toolmodel.TransferChannel, 0)
	for rows.Next() {
		var c toolmodel.TransferChannel
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt, &c.UpdatedAt, &c.FileCount, &c.TextCount); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r *TransferRepository) RenameChannel(ctx context.Context, userID, id int64, name string) (*toolmodel.TransferChannel, error) {
	c := &toolmodel.TransferChannel{ID: id, Name: name}
	err := r.db.QueryRow(ctx, `UPDATE transfer_channels SET name=$3,updated_at=NOW()
 WHERE user_id=$1 AND id=$2 AND NOT deleted RETURNING created_at,updated_at`, userID, id, name).Scan(&c.CreatedAt, &c.UpdatedAt)
	return c, transferDBError(err)
}

// Serialize channel deletion and item creation without holding a lock during upload.
func (r *TransferRepository) locked(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(74672616)`); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return transferDBError(err)
	}
	return tx.Commit(ctx)
}

func (r *TransferRepository) DeleteChannel(ctx context.Context, userID, id int64) error {
	return r.locked(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE transfer_channels SET deleted=TRUE,updated_at=NOW() WHERE user_id=$1 AND id=$2 AND NOT deleted`, userID, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return tool.ErrTransferNotFound
		}
		// Pending uploads finish or fail before cleanup can remove their files.
		_, err = tx.Exec(ctx, `UPDATE transfer_items SET state='deleting' WHERE channel_id=$1 AND state='active'`, id)
		return err
	})
}

func (r *TransferRepository) CreateItem(ctx context.Context, userID int64, i *toolmodel.TransferItem) error {
	return r.locked(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM transfer_channels WHERE id=$1 AND user_id=$2 AND NOT deleted)`, i.ChannelID, userID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return tool.ErrTransferNotFound
		}
		return tx.QueryRow(ctx, `INSERT INTO transfer_items(channel_id,kind,body,storage_key,filename,size_bytes,state)
  VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,created_at`, i.ChannelID, i.Kind, i.Body, i.StorageKey, i.Filename, i.SizeBytes, i.State).Scan(&i.ID, &i.CreatedAt)
	})
}

func (r *TransferRepository) Activate(ctx context.Context, id, size int64) error {
	return r.locked(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE transfer_items i SET state='active',size_bytes=$2,created_at=NOW()
  FROM transfer_channels c WHERE i.id=$1 AND i.channel_id=c.id AND NOT c.deleted AND i.state='pending' AND $2::BIGINT<=$3::BIGINT`, id, size, tool.TransferFileLimit)
		if err == nil && tag.RowsAffected() == 0 {
			return tool.ErrTransferNotFound
		}
		return err
	})
}

func (r *TransferRepository) Abandon(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `UPDATE transfer_items SET state='deleting' WHERE id=$1 AND state='pending'`, id)
	return err
}

func (r *TransferRepository) ListItems(ctx context.Context, userID, channelID int64, kind string, page, pageSize int) ([]toolmodel.TransferItem, error) {
	// Keep the channel existence check and page in one snapshot.
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var id int64
	if err = tx.QueryRow(ctx, `SELECT id FROM transfer_channels WHERE id=$1 AND user_id=$2 AND NOT deleted`, channelID, userID).Scan(&id); err != nil {
		return nil, transferDBError(err)
	}
	rows, err := tx.Query(ctx, `SELECT i.id,i.channel_id,i.kind,left(i.body,200),i.filename,i.size_bytes,i.created_at
 FROM transfer_items i WHERE i.channel_id=$1 AND i.state='active' AND i.created_at>NOW()-INTERVAL '360 hours' AND ($2='' OR i.kind=$2)
 ORDER BY i.created_at DESC,i.id DESC LIMIT $3 OFFSET $4`, channelID, kind, pageSize+1, int64(page-1)*int64(pageSize))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]toolmodel.TransferItem, 0)
	for rows.Next() {
		var i toolmodel.TransferItem
		if err = rows.Scan(&i.ID, &i.ChannelID, &i.Kind, &i.Body, &i.Filename, &i.SizeBytes, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return items, tx.Commit(ctx)
}

func (r *TransferRepository) GetItem(ctx context.Context, userID, id int64) (*toolmodel.TransferItem, error) {
	i := &toolmodel.TransferItem{}
	err := r.db.QueryRow(ctx, `SELECT i.id,i.channel_id,i.kind,i.body,i.storage_key,i.filename,i.size_bytes,i.created_at
 FROM transfer_items i JOIN transfer_channels c ON c.id=i.channel_id
 WHERE i.id=$1 AND c.user_id=$2 AND NOT c.deleted AND i.state='active' AND i.created_at>NOW()-INTERVAL '360 hours'`, id, userID).Scan(&i.ID, &i.ChannelID, &i.Kind, &i.Body, &i.StorageKey, &i.Filename, &i.SizeBytes, &i.CreatedAt)
	return i, transferDBError(err)
}

func (r *TransferRepository) DeleteItem(ctx context.Context, userID, id int64) error {
	return r.locked(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE transfer_items i SET state='deleting' FROM transfer_channels c
  WHERE i.channel_id=c.id AND c.user_id=$1 AND NOT c.deleted AND i.id=$2 AND i.state='active' AND i.created_at>NOW()-INTERVAL '360 hours'`, userID, id)
		if err == nil && tag.RowsAffected() == 0 {
			return tool.ErrTransferNotFound
		}
		return err
	})
}

func (r *TransferRepository) Usage(ctx context.Context, userID int64) (int64, error) {
	var size int64
	err := r.db.QueryRow(ctx, `SELECT COALESCE(SUM(i.size_bytes),0)::BIGINT FROM transfer_items i JOIN transfer_channels c ON c.id=i.channel_id WHERE c.user_id=$1`, userID).Scan(&size)
	return size, err
}

func (r *TransferRepository) CleanupItems(ctx context.Context) ([]toolmodel.TransferItem, error) {
	// Stale pending uploads are older than the ten-minute HTTP transfer deadline.
	_, err := r.db.Exec(ctx, `UPDATE transfer_items SET state='deleting' WHERE id IN (
 SELECT id FROM transfer_items WHERE (state='pending' AND created_at<NOW()-INTERVAL '20 minutes')
 OR (state='active' AND created_at<=NOW()-INTERVAL '360 hours') ORDER BY created_at,id LIMIT 200)`)
	if err != nil {
		return nil, err
	}
	// Rotate failed deletions so they cannot starve later batches.
	rows, err := r.db.Query(ctx, `UPDATE transfer_items SET cleanup_attempted_at=NOW() WHERE id IN (
 SELECT id FROM transfer_items WHERE state='deleting' ORDER BY cleanup_attempted_at,id
 LIMIT 200 FOR UPDATE SKIP LOCKED) RETURNING id,storage_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]toolmodel.TransferItem, 0)
	for rows.Next() {
		var i toolmodel.TransferItem
		if err = rows.Scan(&i.ID, &i.StorageKey); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *TransferRepository) Purge(ctx context.Context, id int64) error {
	return r.locked(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM transfer_items WHERE id=$1 AND state='deleting'`, id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM transfer_channels c WHERE c.deleted AND NOT EXISTS(SELECT 1 FROM transfer_items i WHERE i.channel_id=c.id)`)
		return err
	})
}
