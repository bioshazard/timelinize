package twitter

import (
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

func entityFor(text, short, expanded string) urlEntity {
	start := strings.Index(text, short)
	return urlEntity{URL: short, ExpandedURL: expanded, Indices: []transInt{
		transInt(len(utf16.Encode([]rune(text[:start])))),
		transInt(len(utf16.Encode([]rune(text[:start+len(short)])))),
	}}
}

func TestTweetTextUnicodeEntities(t *testing.T) {
	tests := []struct {
		name, text, want string
		entities         []urlEntity
		media            []*mediaItem
	}{
		{name: "accent and supplementary emoji", text: "Café 😀 https://t.co/a", want: "Café 😀 https://example.test/a", entities: []urlEntity{{URL: "https://t.co/a", ExpandedURL: "https://example.test/a", Indices: []transInt{8, 22}}}},
		{name: "escaped source before URL", text: "a &amp; b https://t.co/a", want: "a & b https://example.test/?a=1&b=2", entities: []urlEntity{{URL: "https://t.co/a", ExpandedURL: "https://example.test/?a=1&amp;b=2", Indices: []transInt{10, 24}}}},
		{name: "missing indices", text: "x https://t.co/a", want: "x https://t.co/a", entities: []urlEntity{{URL: "https://t.co/a", ExpandedURL: "https://example.test/a"}}},
		{name: "negative", text: "x https://t.co/a", want: "x https://t.co/a", entities: []urlEntity{{URL: "https://t.co/a", ExpandedURL: "https://example.test/a", Indices: []transInt{-1, 3}}}},
		{name: "huge out of bounds", text: "x https://t.co/a", want: "x https://t.co/a", entities: []urlEntity{{URL: "https://t.co/a", ExpandedURL: "https://example.test/a", Indices: []transInt{2, 1 << 62}}}},
		{name: "inverted", text: "x https://t.co/a", want: "x https://t.co/a", entities: []urlEntity{{URL: "https://t.co/a", ExpandedURL: "https://example.test/a", Indices: []transInt{15, 2}}}},
		{name: "surrogate midpoint", text: "😀 https://t.co/a", want: "😀 https://t.co/a", entities: []urlEntity{{URL: "https://t.co/a", ExpandedURL: "https://example.test/a", Indices: []transInt{1, 16}}}},
		{name: "stale range", text: "words https://t.co/a", want: "words https://t.co/a", entities: []urlEntity{{URL: "https://t.co/a", ExpandedURL: "https://example.test/a", Indices: []transInt{0, 14}}}},
		{name: "empty expansion", text: "https://t.co/a", want: "https://t.co/a", entities: []urlEntity{{URL: "https://t.co/a", Indices: []transInt{0, 14}}}},
		{name: "text fallback", text: "unlinked &amp; 🦊", want: "unlinked & 🦊"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tw := tweet{Text: tt.text, TweetIDStr: "unchanged-id", createdAtParsed: time.Unix(100, 0), Entities: &twitterEntities{URLs: tt.entities}, ExtendedEntities: &extendedEntities{Media: tt.media}}
			before := tw
			if got := tw.text(); got != tt.want {
				t.Fatalf("text()=%q; want %q", got, tt.want)
			}
			if !reflect.DeepEqual(tw, before) {
				t.Fatal("rendering mutated source tweet")
			}
		})
	}
}

func TestTweetTextMultipleAndMediaEntities(t *testing.T) {
	text := "😀 https://t.co/a then https://t.co/b media https://t.co/c"
	a := entityFor(text, "https://t.co/a", "https://example.test/longer/a")
	b := entityFor(text, "https://t.co/b", "https://example.test/b")
	c := entityFor(text, "https://t.co/c", "https://example.test/media")
	tw := tweet{FullText: text, Entities: &twitterEntities{URLs: []urlEntity{b, a, c}}, ExtendedEntities: &extendedEntities{Media: []*mediaItem{{URL: c.URL, ExpandedURL: c.ExpandedURL, Indices: c.Indices}}}}
	want := "😀 https://example.test/longer/a then https://example.test/b media https://example.test/media"
	if got := tw.text(); got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}

func TestTweetTextRepeatedShortURL(t *testing.T) {
	const text = "https://t.co/a https://t.co/a"
	tw := tweet{FullText: text, Entities: &twitterEntities{URLs: []urlEntity{
		{URL: "https://t.co/a", ExpandedURL: "https://example.test/first", Indices: []transInt{0, 14}},
		{URL: "https://t.co/a", ExpandedURL: "https://example.test/second", Indices: []transInt{15, 29}},
	}}}
	if got, want := tw.text(), "https://example.test/first https://example.test/second"; got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}

func TestTweetTextConflictingEntities(t *testing.T) {
	const text = "https://t.co/a"
	a := entityFor(text, text, "https://example.test/first")
	b := entityFor(text, text, "https://example.test/second")
	tw := tweet{FullText: text, Entities: &twitterEntities{URLs: []urlEntity{a, b}}}
	if got := tw.text(); got != text {
		t.Fatalf("conflicting entities changed source: %q", got)
	}
}

func TestTweetTextRetweetUsesNestedEntities(t *testing.T) {
	const inner = "😀 https://t.co/a &amp; hello"
	entity := entityFor(inner, "https://t.co/a", "https://example.test/a")
	child := &tweet{FullText: inner, User: &twitterUser{ScreenName: "author"}, Entities: &twitterEntities{URLs: []urlEntity{entity}}}
	tw := tweet{FullText: "RT @author truncated", RetweetedStatus: child, Entities: &twitterEntities{URLs: []urlEntity{{URL: "RT", ExpandedURL: "WRONG", Indices: []transInt{0, 2}}}}}
	if got, want := tw.text(), "RT @author 😀 https://example.test/a & hello"; got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
	child.User = nil
	if got, want := tw.text(), "😀 https://example.test/a & hello"; got != want {
		t.Fatalf("missing retweet user: got %q; want %q", got, want)
	}
}

func TestUTF16ByteOffset(t *testing.T) {
	const text = "é😀z"
	for _, tc := range []struct {
		offset transInt
		pos    int
		ok     bool
	}{{-1, 0, false}, {0, 0, true}, {1, 2, true}, {2, 0, false}, {3, 6, true}, {4, 7, true}, {5, 7, false}} {
		pos, ok := utf16ByteOffset(text, tc.offset)
		if ok != tc.ok || ok && pos != tc.pos {
			t.Errorf("offset %d got (%d,%v); want (%d,%v)", tc.offset, pos, ok, tc.pos, tc.ok)
		}
		if ok && !utf8.ValidString(text[:pos]) {
			t.Errorf("offset %d split UTF8", tc.offset)
		}
	}
}

func TestTweetTextOverlappingEntities(t *testing.T) {
	const text = "https://t.co/a"
	a := entityFor(text, text, "https://example.test/outer")
	b := entityFor(text, "t.co", "https://example.test/inner")
	tw := tweet{FullText: text, Entities: &twitterEntities{URLs: []urlEntity{a, b}}, ExtendedEntities: &extendedEntities{Media: []*mediaItem{nil}}}
	if got := tw.text(); got != text {
		t.Fatalf("overlapping entities changed source: %q", got)
	}
}

func TestTweetTextObservedCodePointIndices(t *testing.T) {
	const text = "🧪 Café https://t.co/a then 😀 https://t.co/b"
	for _, codePoints := range []bool{false, true} {
		a := entityFor(text, "https://t.co/a", "https://example.test/a")
		b := entityFor(text, "https://t.co/b", "https://example.test/b")
		if codePoints {
			for _, e := range []*urlEntity{&a, &b} {
				start := strings.Index(text, e.URL)
				e.Indices = []transInt{transInt(len([]rune(text[:start]))), transInt(len([]rune(text[:start+len(e.URL)])))}
			}
		}
		tw := tweet{FullText: text, Entities: &twitterEntities{URLs: []urlEntity{a}}, ExtendedEntities: &extendedEntities{Media: []*mediaItem{{URL: b.URL, ExpandedURL: b.ExpandedURL, Indices: b.Indices}}}}
		if got, want := tw.text(), "🧪 Café https://example.test/a then 😀 https://example.test/b"; got != want {
			t.Fatalf("codePoints=%v got %q; want %q", codePoints, got, want)
		}
	}
}

func TestRuneByteOffset(t *testing.T) {
	const text = "é😀z"
	for _, tc := range []struct {
		offset transInt
		pos    int
		ok     bool
	}{{-1, 0, false}, {0, 0, true}, {1, 2, true}, {2, 6, true}, {3, 7, true}, {4, 7, false}} {
		pos, ok := runeByteOffset(text, tc.offset)
		if ok != tc.ok || ok && pos != tc.pos {
			t.Errorf("offset %d got (%d,%v)", tc.offset, pos, ok)
		}
	}
}

func TestTweetTextAmbiguousIndexProfilesRefused(t *testing.T) {
	text := strings.Repeat("😀", 15) + "https://t.co/a https://t.co/a"
	tw := tweet{FullText: text, Entities: &twitterEntities{URLs: []urlEntity{{URL: "https://t.co/a", ExpandedURL: "https://example.test/a", Indices: []transInt{30, 44}}}}}
	if got := tw.text(); got != text {
		t.Fatal("ambiguous index profiles changed an occurrence")
	}
}
