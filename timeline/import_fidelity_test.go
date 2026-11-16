package timeline

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func fidelityDB(t *testing.T) (*Timeline, *sql.Tx) {
	t.Helper()
	db, err := openAndProvisionTimelineDB(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	tx, err := db.WritePool.Begin()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	return &Timeline{db: db}, tx
}

func TestNoIDLookupDoesNotMatchUnrelatedTombstones(t *testing.T) {
	tl, tx := fidelityDB(t)
	if _, err := tx.Exec(`INSERT INTO items (deleted, initial_content_hash) VALUES (1,?)`, []byte("old")); err != nil {
		t.Fatal(err)
	}
	source := "facebook"
	text := "new text"
	it := &Item{dataText: &text, contentHash: []byte("new")}
	_, err := tl.loadItemRow(context.Background(), tx, 0, 0, it, &source, map[string]bool{"data": true}, true)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unrelated tombstone matched: %v", err)
	}
	it.contentHash = nil
	_, err = tl.loadItemRow(context.Background(), tx, 0, 0, it, &source, map[string]bool{"data": true}, true)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("absent hashes matched: %v", err)
	}
	it.contentHash = []byte("old")
	ir, err := tl.loadItemRow(context.Background(), tx, 0, 0, it, &source, map[string]bool{"data": true}, true)
	if err != nil || ir.Deleted == nil {
		t.Fatalf("real remembered deletion missing: %+v %v", ir, err)
	}
}

func TestStrictOccurrenceLookupAndBinaryText(t *testing.T) {
	tl, tx := fidelityDB(t)
	if _, err := tx.Exec(`INSERT INTO items (data_text,intermediate_location) VALUES ('Hello','messages/a.json#/0'),('hello','messages/a.json#/1'),('Hello','messages/a.json#/2')`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		text, location string
		id             uint64
	}{{"Hello", "messages/a.json#/0", 1}, {"hello", "messages/a.json#/1", 2}, {"Hello", "messages/a.json#/2", 3}, {"HELLO", "messages/a.json#/0", 0}} {
		text := tc.text
		it := &Item{dataText: &text, IntermediateLocation: tc.location}
		ir, err := tl.loadItemRow(context.Background(), tx, 0, 0, it, nil, map[string]bool{"data": true, "intermediate_location": true}, true)
		if tc.id == 0 {
			if !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("case-folded identity: %+v %v", ir, err)
			}
			continue
		}
		if err != nil || ir.ID != tc.id {
			t.Fatalf("occurrence %q got %+v %v", tc.location, ir, err)
		}
	}
}

func TestRememberedContentIsScopedToOccurrence(t *testing.T) {
	source := "facebook"
	otherSource := "twitter"
	text := "same"
	now := time.Unix(100, 0)
	makeItem := func(location string, source *string) *Item {
		it := &Item{Timestamp: now, dataText: &text, IntermediateLocation: location}
		it.makeContentHash()
		it.scopeContentHash(source, map[string]bool{"intermediate_location": true})
		return it
	}
	first := makeItem("messages/a.json#/0", &source)
	replay := makeItem("messages/a.json#/0", &source)
	repeat := makeItem("messages/a.json#/1", &source)
	other := makeItem("messages/a.json#/0", &otherSource)
	if !bytes.Equal(first.contentHash, replay.contentHash) || bytes.Equal(first.contentHash, repeat.contentHash) || bytes.Equal(first.contentHash, other.contentHash) {
		t.Fatal("occurrence/source scope or replay stability lost")
	}
	tl, tx := fidelityDB(t)
	if _, err := tx.Exec(`INSERT INTO items (deleted, initial_content_hash) VALUES (1,?)`, first.contentHash); err != nil {
		t.Fatal(err)
	}
	_, err := tl.loadItemRow(context.Background(), tx, 0, 0, repeat, &source, map[string]bool{"data": true, "intermediate_location": true}, true)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("independent repeat suppressed: %v", err)
	}
	ir, err := tl.loadItemRow(context.Background(), tx, 0, 0, replay, &source, map[string]bool{"data": true, "intermediate_location": true}, true)
	if err != nil || ir.Deleted == nil {
		t.Fatalf("same occurrence deletion forgotten: %+v %v", ir, err)
	}
}

func TestRetrievalPoliciesOnlyApplyToExactKey(t *testing.T) {
	p := &processor{ij: &ImportJob{}}
	for _, same := range []bool{true, false} {
		it := &Item{Retrieval: ItemRetrieval{FieldUpdatePolicies: map[string]FieldUpdatePolicy{"attribute_id": UpdatePolicyPreferIncoming, "classification_id": UpdatePolicyPreferIncoming, "timestamp": UpdatePolicyPreferIncoming}}}
		it.Retrieval.SetKey("media/one.jpg")
		row := ItemRow{RetrievalKey: []byte("different")}
		if same {
			row.RetrievalKey = append([]byte(nil), it.Retrieval.key...)
		}
		if err := p.distillUpdatePolicies(it, row); err != nil {
			t.Fatal(err)
		}
		if same && len(it.fieldUpdatePolicies) != 3 || !same && len(it.fieldUpdatePolicies) != 0 {
			t.Fatalf("same=%v policies=%v", same, it.fieldUpdatePolicies)
		}
	}
}

func TestConversationHashDoesNotMutateParticipants(t *testing.T) {
	participants := uint64Slice{3, 1, 2}
	if participants.hash() != "1,2,3," || participants[0] != 3 || participants[1] != 1 || participants[2] != 2 {
		t.Fatal("hash mutated participants")
	}
}

func TestRecentConversationsHonorSource(t *testing.T) {
	tl, tx := fidelityDB(t)
	exec := func(query string) {
		t.Helper()
		if _, err := tx.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO data_sources (id,name,title,description,standard) VALUES (800,'facebook','FB','test',1),(801,'twitter','Twitter','test',1)`)
	exec(`INSERT INTO entity_types (id,name) VALUES (800,'test_person')`)
	exec(`INSERT INTO entities (id,type_id,name) VALUES (800,800,'Alice'),(801,800,'Bob'),(802,800,'Carol')`)
	exec(`INSERT INTO attributes (id,name,value) VALUES (800,'test','alice'),(801,'test','bob'),(802,'test','carol')`)
	exec(`INSERT INTO entity_attributes (entity_id,attribute_id) VALUES (800,800),(801,801),(802,802)`)
	exec(`INSERT INTO items (id,data_source_id,attribute_id,timestamp,data_text) VALUES (800,800,800,1000,'FB message'),(801,801,800,2000,'Twitter message')`)
	exec(`INSERT INTO relations (id,label,directed,subordinating) VALUES (800,'test_sent',1,0)`)
	exec(`INSERT INTO relationships (relation_id,from_item_id,to_attribute_id) VALUES (800,800,801),(800,801,802)`)
	tl.classifications = map[string]uint64{}
	tl.relations = map[string]uint64{RelSent.Label: 800}
	for _, source := range []string{"facebook", "twitter", "missing"} {
		convs, err := tl.loadRecentConversations(context.Background(), tx, ItemSearchParams{DataSourceName: []string{source}, Limit: 30})
		if err != nil {
			t.Fatal(err)
		}
		if source == "missing" {
			if len(convs) != 0 {
				t.Fatal("unexpected conversation")
			}
			continue
		}
		if len(convs) != 1 {
			t.Fatalf("source %s: %d conversations", source, len(convs))
		}
		for _, c := range convs {
			for _, m := range c.RecentMessages {
				if m.DataSourceName == nil || *m.DataSourceName != source {
					t.Fatalf("source %s leaked %+v", source, m)
				}
			}
		}
	}
}
