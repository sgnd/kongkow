package dest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sgnd/kongkow/internal/store"
)

func fakeTelegram(t *testing.T, reply string) *map[string]any {
	t.Helper()
	got := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/botTOKEN/") {
			t.Errorf("path %q", r.URL.Path)
		}
		got["method"] = strings.TrimPrefix(r.URL.Path, "/botTOKEN/")
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	old := telegramBase
	telegramBase = srv.URL
	t.Cleanup(func() { telegramBase = old })
	return &got
}

func tokenResolver(ref string) (string, error) { return "TOKEN", nil }

func tgDest() *store.Dest {
	return &store.Dest{Name: "tg", Kind: store.DestTelegram, Target: "42", SecretRef: "secret:tg"}
}

func TestTelegramPublish(t *testing.T) {
	got := fakeTelegram(t, `{"ok":true,"result":{}}`)
	r, err := Publish(t.Context(), tgDest(), Content{Name: "n", Title: "Hi", Body: "world"}, tokenResolver)
	if err != nil {
		t.Fatal(err)
	}
	if r.Detail != "sent message" {
		t.Fatalf("detail %q", r.Detail)
	}
	g := *got
	if g["method"] != "sendMessage" || g["chat_id"] != "42" || g["text"] != "Hi\n\nworld" {
		t.Fatalf("request %v", g)
	}
}

func TestTelegramError(t *testing.T) {
	fakeTelegram(t, `{"ok":false,"description":"chat not found"}`)
	_, err := Publish(t.Context(), tgDest(), Content{Name: "n", Body: "x"}, tokenResolver)
	if err == nil || !strings.Contains(err.Error(), "chat not found") {
		t.Fatalf("err %v", err)
	}
}

func TestTelegramNeedsTargetAndSecret(t *testing.T) {
	d := tgDest()
	d.Target = ""
	if _, err := Publish(t.Context(), d, Content{Body: "x"}, tokenResolver); err == nil {
		t.Fatal("want error without target")
	}
	d = tgDest()
	d.SecretRef = ""
	if _, err := Publish(t.Context(), d, Content{Body: "x"}, tokenResolver); err == nil {
		t.Fatal("want error without secret ref")
	}
}
