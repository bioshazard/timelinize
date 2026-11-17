package tlzapp

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"testing"
	"testing/fstest"
)

func TestFrontendAssetFingerprintChangesWithContent(t *testing.T) {
	files := fstest.MapFS{"resources/js/example.js": {Data: []byte("console.log('old');")}}
	app := &App{server: server{frontend: files}}
	render := func() string {
		t.Helper()
		buf := bytes.NewBufferString(`<script src="/resources/js/example.js?v={{include ` + "`/resources/js/example.js`" + ` | sha256sum}}"></script>`)
		if err := executeTemplateInBuffer("index", buf, nil, app); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	before := render()
	if got := render(); got != before {
		t.Fatal("unchanged assets must retain their cache URL")
	}
	files["resources/js/example.js"].Data = []byte("console.log('new');")
	after := render()
	if after == before {
		t.Fatal("edited assets reused the old cache URL")
	}
	want := fmt.Sprintf("v=%x", sha256.Sum256(files["resources/js/example.js"].Data))
	if !bytes.Contains([]byte(after), []byte(want)) {
		t.Fatalf("asset URL does not describe its served bytes: %s", after)
	}
}

func TestFrontendIndexRendersFingerprintedApplicationScripts(t *testing.T) {
	app := &App{server: server{frontend: os.DirFS("../frontend")}}
	data, err := os.ReadFile("../frontend/index.html")
	if err != nil {
		t.Fatal(err)
	}
	buf := bytes.NewBuffer(data)
	if err := executeTemplateInBuffer("index", buf, nil, app); err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`src="/resources/js/entity\.js\?v=[a-f0-9]{64}"`)
	if !pattern.Match(buf.Bytes()) {
		t.Fatal("real application entrypoint omitted the entity script fingerprint")
	}
	if bytes.Contains(buf.Bytes(), []byte("?v={{")) {
		t.Fatal("unrendered asset fingerprint in application HTML")
	}
}
