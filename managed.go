package telebot

import "encoding/json"

// ManagedBotCreated describes a bot created to be managed by the current bot.
type ManagedBotCreated struct {
	Bot *User `json:"bot"`
}

// ManagedBotUpdated describes the creation, token update, or owner update of a managed bot.
type ManagedBotUpdated struct {
	// User that created the bot.
	User *User `json:"user"`
	Bot  *User `json:"bot"`
}

// KeyboardButtonRequestManagedBot defines a request to create a managed bot.
// The result is delivered in a managed_bot update and a managed_bot_created service message.
type KeyboardButtonRequestManagedBot struct {
	// Signed 32-bit identifier, unique within the message.
	RequestID int32 `json:"request_id"`

	// Optional suggested name and username for the bot.
	SuggestedName     string `json:"suggested_name,omitempty"`
	SuggestedUsername string `json:"suggested_username,omitempty"`
}

// BotAccessSettings describes who can access a bot. Its owner always has access.
type BotAccessSettings struct {
	IsAccessRestricted bool   `json:"is_access_restricted"`
	AddedUsers         []User `json:"added_users,omitempty"`
}

// GetManagedBotToken returns the token of the managed bot identified by userID.
func (b *Bot) GetManagedBotToken(userID int64) (string, error) {
	return b.managedBotToken("getManagedBotToken", userID)
}

// ReplaceManagedBotToken revokes the managed bot's current token and returns a new one.
func (b *Bot) ReplaceManagedBotToken(userID int64) (string, error) {
	return b.managedBotToken("replaceManagedBotToken", userID)
}

func (b *Bot) managedBotToken(method string, userID int64) (string, error) {
	data, err := b.Raw(method, map[string]int64{"user_id": userID})
	if err != nil {
		return "", err
	}

	var resp struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", wrapError(err)
	}
	return resp.Result, nil
}

// GetManagedBotAccessSettings returns the access settings of the managed bot identified by userID.
func (b *Bot) GetManagedBotAccessSettings(userID int64) (*BotAccessSettings, error) {
	data, err := b.Raw("getManagedBotAccessSettings", map[string]int64{"user_id": userID})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result *BotAccessSettings `json:"result"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, wrapError(err)
	}
	return resp.Result, nil
}

// SetManagedBotAccessSettings changes who can access a managed bot.
// When restricted, up to 10 addedUserIDs may access it in addition to its owner.
// Telegram ignores addedUserIDs when restricted is false.
func (b *Bot) SetManagedBotAccessSettings(userID int64, restricted bool, addedUserIDs ...int64) error {
	params := map[string]interface{}{
		"user_id":              userID,
		"is_access_restricted": restricted,
	}
	if len(addedUserIDs) > 0 {
		params["added_user_ids"] = addedUserIDs
	}

	data, err := b.Raw("setManagedBotAccessSettings", params)
	if err != nil {
		return err
	}

	var resp struct {
		Result bool `json:"result"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return wrapError(err)
	}
	return nil
}
