package profile

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/openpak/x360-common/pgstate"
)

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("X360_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("X360_TEST_DATABASE_URL not set")
	}
	return dsn
}

// pgSchema returns a private schema name, dropped when the test ends.
func pgSchema(t *testing.T) string {
	dsn := testDSN(t)
	b := make([]byte, 6)
	rand.Read(b)
	schema := "t_profile_" + hex.EncodeToString(b)
	t.Cleanup(func() {
		if p, err := pgstate.Open(context.Background(), dsn, schema); err == nil {
			p.Pool().Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
			p.Close()
		}
	})
	return schema
}

// testPictures opens (or reopens, a restart) the store in schema.
func testPictures(t *testing.T, schema string, key []byte) *PictureStore {
	t.Helper()
	s, err := OpenPictureStore(context.Background(), testDSN(t), schema, "", key)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestOpenRefusesWithoutDSN(t *testing.T) {
	if _, err := OpenPictureStore(context.Background(), "", Schema, "", nil); err == nil {
		t.Fatal("started without a database URL")
	}
}

func TestLegacyHandleFilesImported(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "4f50f6c3-7deb4f30.json"),
		[]byte(`{"Account":"A","Version":"v1","Size":64}`), 0o600); err != nil {
		t.Fatal(err)
	}
	schema := pgSchema(t)
	s, err := OpenPictureStore(context.Background(), testDSN(t), schema, dir, []byte("k"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ref, err := s.lookup(context.Background(), "4f50f6c3-7deb4f30")
	if err != nil || ref.Account != "A" || ref.Size != 64 {
		t.Fatalf("%+v %v", ref, err)
	}
}

func TestEraseDeletesOnlyThatAccount(t *testing.T) {
	ctx := context.Background()
	s := testPictures(t, pgSchema(t), []byte("k"))
	if _, err := s.handles(ctx, "acct-a", "v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.handles(ctx, "acct-b", "v1"); err != nil {
		t.Fatal(err)
	}
	n, err := s.Erase(ctx, "acct-a")
	if err != nil || n != 2 {
		t.Fatalf("erase: %d %v", n, err)
	}
	if n, err := s.Erase(ctx, "acct-a"); err != nil || n != 0 {
		t.Fatalf("second erase: %d %v", n, err)
	}
	var left int
	if err := s.PG.Pool().QueryRow(ctx, `SELECT count(*) FROM pictures WHERE account='acct-b'`).Scan(&left); err != nil || left != 2 {
		t.Fatalf("other account: %d %v", left, err)
	}
}
