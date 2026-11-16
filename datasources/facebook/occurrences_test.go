package facebook

import (
	"context"
	"io"
	"testing"
	"testing/fstest"
	"time"

	"github.com/timelinize/timelinize/timeline"
	"go.uber.org/zap"
)

func itemText(t *testing.T, item *timeline.Item) string {
	t.Helper()
	if item.Content.Data == nil {
		return ""
	}
	r, err := item.Content.Data(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPostsPreserveOccurrenceLocationsAndAttachmentText(t *testing.T) {
	const name = "posts/your_posts_1.json"
	src := fstest.MapFS{name: &fstest.MapFile{Data: []byte(`[
 {"timestamp":1234,"data":[{"post":"hello"}],"attachments":[{"data":[{"text":"attachment text"},{"external_context":{"url":"https://example.org","name":"link"}}]}]},
 {"timestamp":1234,"data":[{"post":"hello"}]},
 {"timestamp":1234,"data":[{"post":"HELLO"}]}
 ]`)}}
	var firstLocations []string
	for run := 0; run < 2; run++ {
		f, err := src.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		ch := make(chan *timeline.Graph, 3)
		a := Archive{owner: timeline.Entity{Name: "Owner"}}
		err = a.processPostsFile(context.Background(), timeline.DirEntry{FS: src}, name, f, timeline.ImportParams{Pipeline: ch, Log: zap.NewNop()})
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for i, want := range []string{"hello", "hello", "HELLO"} {
			g := <-ch
			if got := itemText(t, g.Item); got != want {
				t.Fatalf("item %d text %q != %q", i, got, want)
			}
			if g.Item.ID != "" {
				t.Fatal("derived position must not become platform ID")
			}
			loc := g.Item.IntermediateLocation
			if loc == "" || seen[loc] {
				t.Fatalf("missing/distinct occurrence %q", loc)
			}
			seen[loc] = true
			if run == 0 {
				firstLocations = append(firstLocations, loc)
			} else if firstLocations[i] != loc {
				t.Fatal("non-deterministic occurrence")
			}
			if g.Item.Owner.Name != "Owner" || !g.Item.Timestamp.Equal(time.Unix(1234, 0)) {
				t.Fatal("owner/time changed")
			}
			if i == 0 {
				if len(g.Edges) != 2 {
					t.Fatalf("lost attachment: %d edges", len(g.Edges))
				}
				if got := itemText(t, g.Edges[0].To.Item); got != "attachment text" {
					t.Fatalf("lost attachment text: %q", got)
				}
				for _, e := range g.Edges {
					if e.To.Item.IntermediateLocation == "" || e.To.Item.IntermediateLocation == loc || seen[e.To.Item.IntermediateLocation] {
						t.Fatal("attachment occurrence not distinct")
					}
					seen[e.To.Item.IntermediateLocation] = true
				}
			}
		}
	}
}

func TestMessageOccurrencesDistinguishThreadsSendersAndRepeats(t *testing.T) {
	const name = "messages/inbox/a/message_1.json"
	const other = "messages/inbox/b/message_1.json"
	payload := []byte(`{"participants":[{"name":"Alice"},{"name":"Bob"}],"messages":[
 {"sender_name":"Alice","timestamp_ms":1234000,"content":"same"},
 {"sender_name":"Bob","timestamp_ms":1234000,"content":"same"},
 {"sender_name":"Alice","timestamp_ms":1234000,"content":"same"},
 {"sender_name":"Alice","timestamp_ms":1234000,"content":"SAME"}
 ]}`)
	src := fstest.MapFS{name: &fstest.MapFile{Data: payload}, other: &fstest.MapFile{Data: payload}}
	var locations []string
	for run := 0; run < 2; run++ {
		ch := make(chan *timeline.Graph, 8)
		if err := GetMessages("facebook", timeline.DirEntry{FS: src}, timeline.ImportParams{Pipeline: ch, Log: zap.NewNop()}); err != nil {
			t.Fatal(err)
		}
		if len(ch) != 8 {
			t.Fatalf("lost message graphs: %d", len(ch))
		}
		seen := map[string]bool{}
		for i := 0; i < 8; i++ {
			g := <-ch
			loc := g.Item.IntermediateLocation
			if loc == "" || seen[loc] {
				t.Fatalf("missing/distinct message occurrence: %q", loc)
			}
			seen[loc] = true
			if run == 0 {
				locations = append(locations, loc)
			} else if locations[i] != loc {
				t.Fatal("unstable locations")
			}
			if g.Item.ID != "" {
				t.Fatal("fake platform ID")
			}
			want := "same"
			if i%4 == 3 {
				want = "SAME"
			}
			if itemText(t, g.Item) != want {
				t.Fatal("message text changed")
			}
			actor := "Alice"
			if i%4 == 1 {
				actor = "Bob"
			}
			if g.Item.Owner.Name != actor {
				t.Fatal("wrong sender")
			}
			if len(g.Edges) != 1 || g.Edges[0].To.Entity.Name == actor {
				t.Fatal("recipient lost")
			}
		}
	}
}

func TestMessageAttachmentKeepsURIAndUsesMessageContext(t *testing.T) {
	const name = "messages/inbox/a/message_1.json"
	src := fstest.MapFS{name: &fstest.MapFile{Data: []byte(`{"participants":[{"name":"Alice"},{"name":"Bob"}],"messages":[{"sender_name":"Alice","timestamp_ms":1234000,"content":"photo","photos":[{"uri":"your_activity_across_facebook/media/a.jpg","creation_timestamp":10,"media_metadata":{"photo_metadata":{"exif_data":[{"taken_timestamp":20}]}}}]}]}`)}}
	ch := make(chan *timeline.Graph, 1)
	if err := GetMessages("facebook", timeline.DirEntry{FS: src}, timeline.ImportParams{Pipeline: ch, Log: zap.NewNop()}); err != nil {
		t.Fatal(err)
	}
	g := <-ch
	if len(g.Edges) != 2 {
		t.Fatalf("lost attachment/recipient edges: %d", len(g.Edges))
	}
	attached := g.Edges[0].To.Item
	if attached.IntermediateLocation != "your_activity_across_facebook/media/a.jpg" {
		t.Fatal("media locator changed")
	}
	if attached.Owner.Name != "Alice" || attached.Classification.Name != timeline.ClassMessage.Name || !attached.Timestamp.Equal(time.UnixMilli(1234000)) {
		t.Fatal("message context replaced by media capture context")
	}
	if got, ok := attached.Metadata["Media timestamp"].(time.Time); !ok || !got.Equal(time.Unix(20, 0)) {
		t.Fatal("lost media capture timestamp")
	}
	for _, field := range []string{"attribute_id", "classification_id", "timestamp"} {
		if attached.Retrieval.FieldUpdatePolicies[field] != timeline.UpdatePolicyPreferIncoming {
			t.Fatalf("missing retrieval field preference: %s", field)
		}
	}
	if _, ok := attached.Retrieval.FieldUpdatePolicies["owner"]; ok {
		t.Fatal("invalid owner field policy")
	}
}
