package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
)

// restore replays a dump produced by dump into conn, wrapped in a single
// transaction so a failure leaves the database untouched. Regular statements
// run via Exec; COPY ... FROM stdin blocks stream their data through CopyFrom.
func restore(ctx context.Context, conn pgConn, r io.Reader) error {
	if _, err := conn.Exec(ctx, "BEGIN"); err != nil {
		return fmt.Errorf("begin restore: %w", err)
	}
	if err := replay(ctx, conn, r); err != nil {
		_, _ = conn.Exec(ctx, "ROLLBACK")
		return err
	}
	if _, err := conn.Exec(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit restore: %w", err)
	}
	return nil
}

// replay parses the dump line by line. The dump's own format (see dump) is
// simple enough to split without a full SQL lexer: statements end at a line
// terminating in ';', and COPY data runs from the header line to a lone "\.".
func replay(ctx context.Context, conn pgConn, r io.Reader) error {
	br := bufio.NewReader(r)
	var stmt strings.Builder
	for {
		line, readErr := br.ReadString('\n')
		if len(line) == 0 && readErr != nil {
			break
		}
		raw := strings.TrimRight(line, "\n")
		trimmed := strings.TrimSpace(raw)

		switch {
		case stmt.Len() == 0 && strings.HasPrefix(trimmed, "COPY ") && strings.HasSuffix(trimmed, "FROM stdin;"):
			copySQL := strings.TrimSuffix(trimmed, ";")
			if err := replayCopy(ctx, conn, br, copySQL); err != nil {
				return err
			}
		case stmt.Len() == 0 && (trimmed == "" || strings.HasPrefix(trimmed, "--")):
			// skip blank lines and comments between statements
		default:
			stmt.WriteString(raw)
			stmt.WriteString("\n")
			if strings.HasSuffix(trimmed, ";") {
				sql := strings.TrimSpace(stmt.String())
				stmt.Reset()
				if _, err := conn.Exec(ctx, sql); err != nil {
					return fmt.Errorf("executing %q: %w", truncate(sql), err)
				}
			}
		}

		if readErr != nil {
			break
		}
	}
	return nil
}

// replayCopy collects COPY data up to the terminating "\." line and streams it
// to the server via CopyFrom.
func replayCopy(ctx context.Context, conn pgConn, br *bufio.Reader, copySQL string) error {
	var data bytes.Buffer
	for {
		line, readErr := br.ReadString('\n')
		if len(line) == 0 && readErr != nil {
			return fmt.Errorf("unexpected EOF in COPY data for %q", copySQL)
		}
		if strings.TrimRight(line, "\n") == `\.` {
			break
		}
		data.WriteString(line)
		if readErr != nil {
			return fmt.Errorf("unexpected EOF in COPY data for %q", copySQL)
		}
	}
	if _, err := conn.CopyFrom(ctx, bytes.NewReader(data.Bytes()), copySQL); err != nil {
		return fmt.Errorf("restoring data via %q: %w", copySQL, err)
	}
	return nil
}

// truncate shortens a statement for error messages.
func truncate(s string) string {
	const max = 60
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
