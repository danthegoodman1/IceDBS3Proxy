package icedb

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestReadLog(t *testing.T) {
	i, err := NewIceDBLogReader(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	snap, err := i.ReadState(context.Background(), "tenant", time.Now().UnixMilli())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v, %d", snap, len(snap.AliveFiles))

	snap, err = i.ReadState(context.Background(), "tenant", 0)
	if !errors.Is(err, ErrNoLogFiles) {
		t.Fatal("found files?")
	}
}
