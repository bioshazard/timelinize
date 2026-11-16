package facebook

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/timelinize/timelinize/timeline"
)

// This exercises the registered importer, asynchronous native job, processor,
// SQLite identity, native deletion/merge, and stored bytes, not just emitted graphs.
func TestNativeImportOccurrenceReplayAndMediaContext(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	tl, err := timeline.Create(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	tl.SetObfuscationFunc(func() (timeline.ObfuscationOptions, bool) { return timeline.ObfuscationOptions{}, false })
	defer func() {
		if tl != nil {
			_ = tl.Close()
		}
	}()
	if err := tl.StoreEntity(ctx, timeline.Entity{Type: "person", Name: "Owner"}); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	write := func(name string, data []byte) {
		t.Helper()
		dest := filepath.Join(source, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("profile_information/profile_information.json", []byte(`{"profile_v2":{"username":"synthetic-owner","name":{"full_name":"Owner"}}}`))
	write("posts/your_posts_1.json", []byte(`[
 {"timestamp":1234,"data":[{"post":"same"}],"attachments":[{"data":[{"text":"attachment text"}]}]},
 {"timestamp":1234,"data":[{"post":"same"}]},
 {"timestamp":1234,"data":[{"post":"SAME"}]}
 ]`))
	messages := []byte(`{"participants":[{"name":"Owner"},{"name":"Bob"}],"messages":[
 {"sender_name":"Owner","timestamp_ms":1234000,"content":"same"},
 {"sender_name":"Bob","timestamp_ms":1234000,"content":"same"},
 {"sender_name":"Owner","timestamp_ms":1234000,"content":"same"},
 {"sender_name":"Owner","timestamp_ms":1234000,"content":"SAME"}
 ]}`)
	write("messages/inbox/a/message_1.json", messages)
	write("messages/inbox/b/message_1.json", messages)
	write("messages/archived_threads/c/message_1.json", []byte(`{"participants":[{"name":"Owner"},{"name":"Bob"}],"messages":[{"sender_name":"Bob","timestamp_ms":1234000,"content":"photo","photos":[{"uri":"your_activity_across_facebook/media/a.jpg","creation_timestamp":10,"media_metadata":{"photo_metadata":{"exif_data":[{"taken_timestamp":20}]}}}]},{"sender_name":"Owner","timestamp_ms":1235000,"photos":[{"uri":"your_activity_across_facebook/media/b.jpg","creation_timestamp":30}]}]}`))
	// A real tiny JPEG exercises the media walker as well as the message reference.
	image, err := base64.StdEncoding.DecodeString("/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAP//////////////////////////////////////////////////////////////////////////////////////2wBDAf//////////////////////////////////////////////////////////////////////////////////////wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAX/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCfAAr/2Q==")
	if err != nil {
		t.Fatal(err)
	}
	write("your_activity_across_facebook/media/a.jpg", image)
	write("your_activity_across_facebook/media/b.jpg", image)
	db, err := sql.Open("sqlite3", "file:"+filepath.Join(repo, timeline.DBFilename)+"?mode=ro&_busy_timeout=10000")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	count := func(q string, args ...any) int {
		t.Helper()
		var n int
		if err := db.QueryRow(q, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	run := func() {
		t.Helper()
		id, err := tl.CreateJob(&timeline.ImportJob{Plan: timeline.ImportPlan{Files: []timeline.FileImport{{DataSourceName: "facebook", DataSourceOptions: []byte(`{"username":"synthetic-owner"}`), Filenames: []string{source}}}}, ProcessingOptions: timeline.ProcessingOptions{Integrity: true, BatchSize: 1, ItemUniqueConstraints: map[string]bool{"timestamp": true, "data": true, "data_source_name": true, "classification_name": true, "intermediate_location": true}}}, time.Time{}, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			jobs, err := tl.GetJobs(ctx, []uint64{id}, 0)
			if err != nil {
				t.Fatal(err)
			}
			switch jobs[0].State {
			case timeline.JobSucceeded:
				return
			case timeline.JobFailed, timeline.JobAborted:
				t.Fatalf("import ended: %+v", jobs[0])
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("native import timed out")
	}
	run()
	if n := count("SELECT count(*) FROM items WHERE deleted IS NULL"); n != 15 {
		t.Fatalf("got %d items, want 15", n)
	}
	// Exact data can be inline or MIME-sniffed into files; both are authoritative.
	rows, err := db.Query(`SELECT intermediate_location, data_text, data_file FROM items WHERE deleted IS NULL`)
	if err != nil {
		t.Fatal(err)
	}
	texts := map[string]string{}
	for rows.Next() {
		var loc string
		var text, file sql.NullString
		if err := rows.Scan(&loc, &text, &file); err != nil {
			t.Fatal(err)
		}
		if file.Valid {
			b, e := os.ReadFile(filepath.Join(repo, file.String))
			if e != nil {
				t.Fatal(e)
			}
			text.String = string(b)
		}
		texts[loc] = text.String
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	for _, thread := range []string{"a", "b"} {
		for i, want := range []string{"same", "same", "same", "SAME"} {
			loc := "messages/inbox/" + thread + "/message_1.json#/messages/" + strconv.Itoa(i)
			if texts[loc] != want {
				t.Fatalf("%s: %q != %q", loc, texts[loc], want)
			}
		}
	}
	for i, want := range []string{"same", "same", "SAME"} {
		loc := "posts/your_posts_1.json#/posts/" + strconv.Itoa(i)
		if texts[loc] != want {
			t.Fatalf("%s: %q", loc, texts[loc])
		}
	}
	for _, thread := range []string{"a", "b"} {
		for i, sender := range []string{"Owner", "Bob", "Owner", "Owner"} {
			loc := "messages/inbox/" + thread + "/message_1.json#/messages/" + strconv.Itoa(i)
			if n := count(`SELECT count(*) FROM extended_items i JOIN attributes a ON a.id=i.attribute_id WHERE i.intermediate_location=? AND a.value=? AND i.classification_name='message' AND i.timestamp=1234000 AND i.original_id IS NULL`, loc, sender); n != 1 {
				t.Fatalf("message actor/class/time/native ID mismatch: %s", loc)
			}
		}
	}
	if n := count("SELECT count(*) FROM items WHERE original_id IS NOT NULL"); n != 0 {
		t.Fatal("derived positions became native IDs")
	}
	var mediaMetadata string
	if err := db.QueryRow(`SELECT metadata FROM items WHERE intermediate_location='your_activity_across_facebook/media/a.jpg' AND deleted IS NULL`).Scan(&mediaMetadata); err != nil {
		t.Fatal(err)
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(mediaMetadata), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["Media timestamp"] != "1970-01-01T00:00:20Z" {
		t.Fatalf("lost media timestamp: %v", metadata)
	}
	if texts["posts/your_posts_1.json#/posts/0/attachments/0/data/0"] != "attachment text" {
		t.Fatal("lost post attachment text")
	}
	if texts["your_activity_across_facebook/media/a.jpg"] != string(image) {
		t.Fatal("media bytes changed")
	}
	if n := count(`SELECT count(*) FROM extended_items i JOIN attributes a ON a.id=i.attribute_id WHERE i.intermediate_location='your_activity_across_facebook/media/a.jpg' AND i.deleted IS NULL AND a.value='Bob' AND i.classification_name='message' AND i.timestamp=1234000`); n != 1 {
		t.Fatal("message media owner/class/time lost")
	}
	if n := count(`SELECT count(*) FROM extended_items i JOIN attributes a ON a.id=i.attribute_id WHERE i.intermediate_location='your_activity_across_facebook/media/b.jpg' AND i.deleted IS NULL AND a.value='Owner' AND i.classification_name='message' AND i.timestamp=1235000`); n != 1 {
		t.Fatal("media-only message context lost")
	}
	if n := count(`SELECT count(*) FROM relationships r JOIN relations rel ON rel.id=r.relation_id JOIN items parent ON parent.id=r.from_item_id JOIN items child ON child.id=r.to_item_id WHERE rel.label='attachment' AND ((parent.intermediate_location='posts/your_posts_1.json#/posts/0' AND child.intermediate_location='posts/your_posts_1.json#/posts/0/attachments/0/data/0') OR (parent.intermediate_location='messages/archived_threads/c/message_1.json#/messages/0' AND child.intermediate_location='your_activity_across_facebook/media/a.jpg'))`); n != 2 {
		t.Fatalf("attachment edges %d != 2", n)
	}
	// Every text message retains the other participant as recipient, across both threads.
	if n := count(`SELECT count(*) FROM relationships r JOIN items i ON i.id=r.from_item_id JOIN relations rel ON rel.id=r.relation_id JOIN attributes recipient ON recipient.id=r.to_attribute_id JOIN attributes sender ON sender.id=i.attribute_id WHERE i.intermediate_location LIKE 'messages/inbox/%' AND rel.label='sent' AND recipient.value!=sender.value`); n != 8 {
		t.Fatalf("recipient edges %d != 8", n)
	}
	var sourceOwner uint64
	if err := db.QueryRow(`SELECT ea.entity_id FROM entity_attributes ea JOIN attributes a ON a.id=ea.attribute_id WHERE a.name='facebook_username' AND a.value='synthetic-owner'`).Scan(&sourceOwner); err != nil {
		t.Fatal(err)
	}
	if sourceOwner == 1 {
		t.Fatal("fixture did not create separate source account")
	}
	if err := tl.MergeEntities(ctx, 1, []uint64{sourceOwner}); err != nil {
		t.Fatal(err)
	}
	owner, err := tl.LoadEntity(1)
	if err != nil {
		t.Fatal(err)
	}
	if owner.AttributeValue("facebook_username") != "synthetic-owner" {
		t.Fatal("owner lost source identity")
	}
	if n := count(`SELECT count(*) FROM items i JOIN entity_attributes ea ON ea.attribute_id=i.attribute_id WHERE i.intermediate_location LIKE 'posts/your_posts_1.json#/posts/%' AND ea.entity_id=1`); n != 4 {
		t.Fatalf("owner merge ownership %d != 4", n)
	}
	beforeEdges := count("SELECT count(*) FROM relationships")
	run()
	if count("SELECT count(*) FROM items WHERE deleted IS NULL") != 15 || count("SELECT count(*) FROM relationships") != beforeEdges {
		t.Fatal("replay changed multiplicity/relationships")
	}
	var deleted uint64
	if err := db.QueryRow(`SELECT id FROM items WHERE intermediate_location='posts/your_posts_1.json#/posts/1' AND deleted IS NULL`).Scan(&deleted); err != nil {
		t.Fatal(err)
	}
	if err := tl.DeleteItems(ctx, []uint64{deleted}, timeline.DeleteOptions{Remember: true}); err != nil {
		t.Fatal(err)
	}
	run()
	if n := count("SELECT count(*) FROM items WHERE deleted IS NULL"); n != 14 {
		t.Fatalf("deletion/replay active count %d != 14", n)
	}
	if count(`SELECT count(*) FROM items WHERE intermediate_location='posts/your_posts_1.json#/posts/0' AND deleted IS NULL`) != 1 {
		t.Fatal("independent repeat suppressed")
	}
	// Append a new provenance member after deletion: new identical content must
	// survive the remembered deletion of another occurrence, as must unrelated content.
	write("posts/your_posts_2.json", []byte(`[{"timestamp":1234,"data":[{"post":"same"}]},{"timestamp":1234,"data":[{"post":"unrelated"}]}]`))
	run()
	if n := count("SELECT count(*) FROM items WHERE deleted IS NULL"); n != 16 {
		t.Fatalf("new occurrence/unrelated import count %d != 16", n)
	}
	if err := tl.Close(); err != nil {
		t.Fatal(err)
	}
	tl = nil
	tl, err = timeline.Open(ctx, repo, false)
	if err != nil {
		t.Fatal(err)
	}
	tl.SetObfuscationFunc(func() (timeline.ObfuscationOptions, bool) { return timeline.ObfuscationOptions{}, false })
	result, err := tl.Search(ctx, timeline.ItemSearchParams{DataSourceName: []string{"facebook"}, Astructured: true, OnlyTotal: true})
	if err != nil || result.Total == nil || *result.Total != 16 {
		t.Fatalf("reopened native search: %+v %v", result, err)
	}
	owner, err = tl.LoadEntity(1)
	if err != nil || owner.AttributeValue("facebook_username") != "synthetic-owner" {
		t.Fatalf("reopened owner identity: %+v %v", owner, err)
	}
	var integrity string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("integrity %q: %v", integrity, err)
	}
}

func TestNativeOwnerMergeIntoNilMetadata(t *testing.T) {
	ctx := context.Background()
	tl, err := timeline.Create(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer tl.Close()
	if err := tl.StoreEntity(ctx, timeline.Entity{Type: "person", Name: "Owner"}); err != nil {
		t.Fatal(err)
	}
	owner, err := tl.LoadEntity(1)
	if err != nil {
		t.Fatal(err)
	}
	if owner.Metadata != nil {
		t.Fatal("fixture owner metadata must be nil")
	}
	if err := tl.StoreEntity(ctx, timeline.Entity{Type: "person", Name: "Source", Metadata: timeline.Metadata{"Source label": "synthetic", "Number": 42}, Attributes: []timeline.Attribute{{Name: "synthetic_account", Value: "account-1", Identity: true}}}); err != nil {
		t.Fatal(err)
	}
	if err := tl.MergeEntities(ctx, 1, []uint64{2}); err != nil {
		t.Fatal(err)
	}
	owner, err = tl.LoadEntity(1)
	if err != nil {
		t.Fatal(err)
	}
	if owner.Metadata["Source label"] != "synthetic" || owner.Metadata["Number"] != float64(42) || owner.AttributeValue("synthetic_account") != "account-1" {
		t.Fatalf("merge lost metadata/identity: %+v", owner)
	}
}
