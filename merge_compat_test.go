package telebot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDraftCompatibility(t *testing.T) {
	for _, method := range []string{"SendDraft", "SendMessageDraft"} {
		t.Run(method, func(t *testing.T) {
			var body map[string]string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/bottoken/sendMessageDraft", r.URL.Path)
				body = nil
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Write([]byte(`{"ok":true,"result":true}`))
			}))
			defer srv.Close()

			b, err := NewBot(Settings{Offline: true, Token: "token", URL: srv.URL, Client: srv.Client()})
			require.NoError(t, err)
			call := func(to Recipient, opts ...interface{}) error {
				if method == "SendDraft" {
					return b.SendDraft(to, 42, "hello", opts...)
				}
				ok, err := b.SendMessageDraft(to, 42, "hello", opts...)
				assert.Equal(t, err == nil, ok)
				return err
			}

			assert.Equal(t, ErrBadRecipient, call(nil))
			opts := &SendOptions{ThreadID: 7, ParseMode: ModeHTML}
			require.NoError(t, call(ChatID(123), opts))
			assert.Equal(t, "123", body["chat_id"])
			assert.Equal(t, "42", body["draft_id"])
			assert.Equal(t, "hello", body["text"])
			assert.Equal(t, "7", body["message_thread_id"])
			assert.Equal(t, ModeHTML, body["parse_mode"])

			opts.Entities = Entities{{Type: EntityBold, Offset: 0, Length: 5}}
			require.NoError(t, call(ChatID(123), opts))
			assert.NotContains(t, body, "parse_mode")
			assert.JSONEq(t, `[{"type":"bold","offset":0,"length":5}]`, body["entities"])
		})
	}
}

func TestNotifyBusinessAndThread(t *testing.T) {
	var body map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/bottoken/sendChatAction", r.URL.Path)
		body = nil
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer srv.Close()

	b, err := NewBot(Settings{Offline: true, Token: "token", URL: srv.URL, Client: srv.Client()})
	require.NoError(t, err)
	connectionID := "business-connection"
	require.NoError(t, b.Notify(ChatID(123), Typing, &connectionID, 7))
	assert.Equal(t, "123", body["chat_id"])
	assert.Equal(t, "typing", body["action"])
	assert.Equal(t, connectionID, body["business_connection_id"])
	assert.Equal(t, "7", body["message_thread_id"])

	msg := &Message{Chat: &Chat{ID: 123}, ThreadID: 9}
	for _, update := range []Update{
		{Message: msg},
		{BusinessMessage: msg},
		{EditedBusinessMessage: msg},
	} {
		c := b.NewContext(update)
		assert.Same(t, msg, c.Message())
		require.NoError(t, c.Notify(Typing))
		assert.Equal(t, "123", body["chat_id"])
		assert.Equal(t, "9", body["message_thread_id"])
		assert.NotContains(t, body, "business_connection_id")
	}
}
