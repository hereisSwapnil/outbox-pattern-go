package relay

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
)

type EventHandler func(ctx context.Context, eventType string, payload string) error

type WALReader struct {
	replConnStr  string
	slotName     string
	publication  string
	eventHandler EventHandler
}

func NewWALReader(replConnStr, slotName, publication string, handler EventHandler) *WALReader {
	return &WALReader{
		replConnStr:  replConnStr,
		slotName:     slotName,
		publication:  publication,
		eventHandler: handler,
	}
}

func (r *WALReader) Start(ctx context.Context) error {
	replConn, err := pgconn.Connect(ctx, r.replConnStr)
	if err != nil {
		return fmt.Errorf("repl connect: %w", err)
	}
	defer replConn.Close(ctx)

	err = pglogrepl.StartReplication(ctx, replConn, r.slotName, 0,
		pglogrepl.StartReplicationOptions{
			PluginArgs: []string{
				"proto_version '1'",
				"publication_names '" + r.publication + "'",
			},
		},
	)
	if err != nil {
		return fmt.Errorf("start replication: %w", err)
	}

	log.Println("🔌 Listening on WAL stream...")

	standbyDeadline := time.Now().Add(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if time.Now().After(standbyDeadline) {
			err = pglogrepl.SendStandbyStatusUpdate(ctx, replConn,
				pglogrepl.StandbyStatusUpdate{
					WALWritePosition: 0,
					ReplyRequested:   false,
				},
			)
			if err != nil {
				log.Printf("failed to send standby update: %v", err)
			}
			standbyDeadline = time.Now().Add(10 * time.Second)
		}

		ctx2, cancel := context.WithDeadline(ctx, standbyDeadline)
		rawMsg, err := replConn.ReceiveMessage(ctx2)
		cancel()

		if err != nil {
			if pgconn.Timeout(err) {
				continue
			}
			return fmt.Errorf("receive: %w", err)
		}

		copyData, ok := rawMsg.(*pgproto3.CopyData)
		if !ok {
			continue
		}

		if copyData.Data[0] != pglogrepl.XLogDataByteID {
			continue
		}

		xld, err := pglogrepl.ParseXLogData(copyData.Data[1:])
		if err != nil {
			continue
		}

		msg, err := pglogrepl.Parse(xld.WALData)
		if err != nil {
			continue
		}

		if insert, ok := msg.(*pglogrepl.InsertMessage); ok {
			cols := insert.Tuple.Columns
			if len(cols) >= 3 {
				eventType := string(cols[1].Data)
				payload := string(cols[2].Data)

				err := r.eventHandler(ctx, eventType, payload)
				if err != nil {
					log.Printf("error handling event %s: %v", eventType, err)
				}

				err = pglogrepl.SendStandbyStatusUpdate(ctx, replConn,
					pglogrepl.StandbyStatusUpdate{
						WALWritePosition: xld.WALStart + pglogrepl.LSN(len(xld.WALData)),
					},
				)
				if err != nil {
					log.Printf("failed to send standby update after processing: %v", err)
				}
			}
		}
	}
}
