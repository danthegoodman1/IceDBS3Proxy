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

	snap, err := i.ReadState(context.Background(), "tenant", "", time.Now().UnixMilli(), 1000)
	if err != nil {
		t.Fatal(err)
	}

	// t.Logf("%+v, %d", snap, len(snap.AliveFiles))

	snap, err = i.ReadState(context.Background(), "tenant", "", 0, 1000)
	if !errors.Is(err, ErrNoLogFiles) {
		t.Fatal("found files?")
	}

	t.Log("Checking with limit and offset")
	snap, err = i.ReadState(context.Background(), "tenant", "", time.Now().UnixMilli(), 100)
	if err != nil {
		t.Fatal(err)
	}

	items := []FileMarker{}
	for _, val := range snap.AliveFiles {
		items = append(items, val)
	}

	if len(items) != 100 {
		t.Fatalf("did not limit to 100, got %d", len(items))
	}

	snap, err = i.ReadState(context.Background(), "tenant", items[len(items)-1].Path, time.Now().UnixMilli(), 100)
	if err != nil {
		t.Fatal(err)
	}

	items2 := []FileMarker{}
	for _, val := range snap.AliveFiles {
		items2 = append(items2, val)
	}

	t.Log("Got offsets", items[len(items)-1].Path, items2[0].Path)

	if items2[0].Path <= items[len(items)-1].Path {
		t.Fatal("Second page was not greater than offset")
	}
}
