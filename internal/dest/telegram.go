package dest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sgnd/kongkow/internal/store"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

type tgAPI struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

func telegramGetMe(ctx context.Context, token string) (string, error) {
	var res struct {
		Username string `json:"username"`
	}
	if err := tgCall(ctx, token, "getMe", nil, &res); err != nil {
		return "", err
	}
	return res.Username, nil
}

func publishTelegram(ctx context.Context, d *store.Dest, c Content, resolve ResolveToken) (*Result, error) {
	if d.Target == "" {
		return nil, fmt.Errorf("telegram destination needs --target CHAT_ID")
	}
	if d.SecretRef == "" {
		return nil, fmt.Errorf("telegram destination needs --secret-ref secret:NAME")
	}
	token, err := resolve(d.SecretRef)
	if err != nil {
		return nil, err
	}
	text := c.Body
	if c.Title != "" {
		text = c.Title + "\n\n" + c.Body
	}
	if len(c.Media) > 0 {
		text += "\n\nMedia:\n" + strings.Join(c.Media, "\n")
	}
	if text == "" {
		text = c.Name
	}
	parseMode := ""
	if d.Options != nil {
		parseMode = d.Options["parse_mode"]
	}
	payload := map[string]any{
		"chat_id": d.Target,
		"text":    text,
	}
	if parseMode != "" {
		payload["parse_mode"] = parseMode
	}
	if err := tgCall(ctx, token, "sendMessage", payload, nil); err != nil {
		return nil, err
	}
	return &Result{Dest: d.Name, Kind: d.Kind, Target: d.Target, Detail: "sent message"}, nil
}

func tgCall(ctx context.Context, token, method string, payload any, out any) error {
	url := "https://api.telegram.org/bot" + token + "/" + method
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var api tgAPI
	if err := json.Unmarshal(raw, &api); err != nil {
		return fmt.Errorf("telegram: could not read answer (%w)", err)
	}
	if !api.OK {
		msg := api.Description
		if msg == "" {
			msg = string(raw)
		}
		return fmt.Errorf("telegram: %s", msg)
	}
	if out != nil && len(api.Result) > 0 {
		return json.Unmarshal(api.Result, out)
	}
	return nil
}
