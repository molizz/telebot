package telebot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ API = (*Bot)(nil)

func TestManagedBotMethods(t *testing.T) {
	const botID int64 = 4503599627370495
	tests := []struct {
		name     string
		method   string
		body     string
		response string
		call     func(API) (interface{}, error)
		want     interface{}
	}{
		{
			name: "get token", method: "getManagedBotToken",
			body: `{"user_id":4503599627370495}`, response: `{"ok":true,"result":"managed-token"}`,
			call: func(b API) (interface{}, error) { return b.GetManagedBotToken(botID) },
			want: "managed-token",
		},
		{
			name: "replace token", method: "replaceManagedBotToken",
			body: `{"user_id":4503599627370495}`, response: `{"ok":true,"result":"new-managed-token"}`,
			call: func(b API) (interface{}, error) { return b.ReplaceManagedBotToken(botID) },
			want: "new-managed-token",
		},
		{
			name: "get access", method: "getManagedBotAccessSettings",
			body:     `{"user_id":4503599627370495}`,
			response: `{"ok":true,"result":{"is_access_restricted":true,"added_users":[{"id":4503599627370494,"first_name":"Owner"}]}}`,
			call:     func(b API) (interface{}, error) { return b.GetManagedBotAccessSettings(botID) },
			want:     &BotAccessSettings{IsAccessRestricted: true, AddedUsers: []User{{ID: botID - 1, FirstName: "Owner"}}},
		},
		{
			name: "restricted access", method: "setManagedBotAccessSettings",
			body:     `{"user_id":4503599627370495,"is_access_restricted":true,"added_user_ids":[4503599627370494,123]}`,
			response: `{"ok":true,"result":true}`,
			call:     func(b API) (interface{}, error) { return nil, b.SetManagedBotAccessSettings(botID, true, botID-1, 123) },
		},
		{
			name: "owner only", method: "setManagedBotAccessSettings",
			body:     `{"user_id":4503599627370495,"is_access_restricted":true}`,
			response: `{"ok":true,"result":true}`,
			call:     func(b API) (interface{}, error) { return nil, b.SetManagedBotAccessSettings(botID, true) },
		},
		{
			name: "unrestricted access", method: "setManagedBotAccessSettings",
			body:     `{"user_id":4503599627370495,"is_access_restricted":false}`,
			response: `{"ok":true,"result":true}`,
			call:     func(b API) (interface{}, error) { return nil, b.SetManagedBotAccessSettings(botID, false) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/bottoken/"+tt.method, r.URL.Path)
				var body json.RawMessage
				if assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
					assert.JSONEq(t, tt.body, string(body))
				}
				w.Write([]byte(tt.response))
			}))
			defer srv.Close()
			b, err := NewBot(Settings{Offline: true, Token: "token", URL: srv.URL, Client: srv.Client()})
			require.NoError(t, err)
			result, err := tt.call(b)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestManagedBotMethodErrors(t *testing.T) {
	methods := []struct {
		name string
		call func(*Bot) error
	}{
		{"getManagedBotToken", func(b *Bot) error { _, err := b.GetManagedBotToken(123); return err }},
		{"replaceManagedBotToken", func(b *Bot) error { _, err := b.ReplaceManagedBotToken(123); return err }},
		{"getManagedBotAccessSettings", func(b *Bot) error { _, err := b.GetManagedBotAccessSettings(123); return err }},
		{"setManagedBotAccessSettings", func(b *Bot) error { return b.SetManagedBotAccessSettings(123, false) }},
	}
	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			for _, response := range []string{
				`{"ok":false,"error_code":400,"description":"Bad Request: managed bot not found"}`,
				`invalid json`,
			} {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(response))
				}))
				b, err := NewBot(Settings{Offline: true, URL: srv.URL, Client: srv.Client()})
				require.NoError(t, err)
				assert.Error(t, method.call(b))
				srv.Close()
				assert.Error(t, method.call(b), "transport errors must propagate")
			}
		})
	}

	for _, method := range methods[:3] {
		t.Run(method.name+" invalid result", func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"ok":true,"result":123}`))
			}))
			defer srv.Close()
			b, err := NewBot(Settings{Offline: true, URL: srv.URL, Client: srv.Client()})
			require.NoError(t, err)
			assert.Error(t, method.call(b))
		})
	}
}

func TestManagedBotUpdates(t *testing.T) {
	b, err := NewBot(Settings{Offline: true, Synchronous: true})
	require.NoError(t, err)

	var managed, created int
	b.Handle(OnManagedBot, func(c Context) error {
		managed++
		require.NotNil(t, c.ManagedBot())
		assert.Same(t, c.Update().ManagedBot, c.ManagedBot())
		assert.Equal(t, int64(4503599627370495), c.ManagedBot().Bot.ID)
		assert.True(t, c.ManagedBot().Bot.IsBot)
		require.NotNil(t, c.Sender())
		assert.Equal(t, int64(123), c.Sender().ID)
		assert.Same(t, c.ManagedBot().User, c.Sender())
		assert.Same(t, c.Sender(), c.Recipient())
		assert.Nil(t, c.Message())
		assert.Nil(t, c.Chat())
		return nil
	})
	b.Handle(OnManagedBotCreated, func(c Context) error {
		created++
		require.NotNil(t, c.Message())
		require.NotNil(t, c.Message().ManagedBotCreated)
		assert.Equal(t, int64(4503599627370495), c.Message().ManagedBotCreated.Bot.ID)
		assert.Equal(t, int64(123), c.Sender().ID)
		assert.Same(t, c.Chat(), c.Recipient())
		assert.Nil(t, c.ManagedBot())
		return nil
	})
	b.Handle(OnText, func(c Context) error {
		t.Error("managed bot updates must not trigger text handlers")
		return nil
	})

	for _, data := range []string{
		`{"update_id":1,"managed_bot":{"user":{"id":123},"bot":{"id":4503599627370495,"is_bot":true}}}`,
		`{"update_id":2,"message":{"message_id":42,"chat":{"id":123,"type":"private"},"from":{"id":123},"managed_bot_created":{"bot":{"id":4503599627370495,"is_bot":true}}}}`,
	} {
		var update Update
		require.NoError(t, json.Unmarshal([]byte(data), &update))
		b.ProcessUpdate(update)
	}
	assert.Equal(t, 1, managed)
	assert.Equal(t, 1, created)
	assert.Nil(t, b.NewContext(Update{}).ManagedBot())
	assert.Contains(t, AllowedUpdates, "managed_bot")
}

func TestManagedBotButton(t *testing.T) {
	markup := &ReplyMarkup{}
	request := &KeyboardButtonRequestManagedBot{
		RequestID: -2147483648, SuggestedName: "Helper", SuggestedUsername: "helper_bot",
	}
	btn := markup.ManagedBot("Create a bot", request)
	assert.Nil(t, btn.Inline())
	require.NotNil(t, btn.Reply())
	assert.Same(t, request, btn.Reply().ManagedBot)
	markup.Reply(markup.Row(btn))
	data, err := json.Marshal(markup)
	require.NoError(t, err)
	assert.JSONEq(t, `{"keyboard":[[{"text":"Create a bot","request_managed_bot":{"request_id":-2147483648,"suggested_name":"Helper","suggested_username":"helper_bot"}}]]}`, string(data))

	data, err = json.Marshal(KeyboardButtonRequestManagedBot{})
	require.NoError(t, err)
	assert.JSONEq(t, `{"request_id":0}`, string(data))
	data, err = json.Marshal(markup.Text("Plain").Reply())
	require.NoError(t, err)
	assert.JSONEq(t, `{"text":"Plain"}`, string(data))
	assert.Panics(t, func() { markup.Inline(markup.Row(btn)) })
}

func TestManagedBotCapability(t *testing.T) {
	var user User
	require.NoError(t, json.Unmarshal([]byte(`{"id":123,"is_bot":true,"can_manage_bots":true}`), &user))
	assert.True(t, user.CanManageBots)
}
