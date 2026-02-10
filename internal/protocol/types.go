package protocol

const ProtocolVersion = 1

// Message types
const (
	TypeAuth       = "auth"
	TypeAuthOk     = "auth.ok"
	TypeAuthFail   = "auth.fail"
	TypeChatSend   = "chat.send"
	TypeChatStream = "chat.stream"
	TypeChatDone   = "chat.done"
	TypeChatError  = "chat.error"
	TypePing        = "ping"
	TypePong        = "pong"
	TypeStatus      = "status"
	TypeKeyExchange = "key_exchange"

	// Agent management
	TypeAgentList       = "agent.list"
	TypeAgentListResult = "agent.list.result"
	TypeAgentCreate     = "agent.create"
	TypeAgentUpdate     = "agent.update"
	TypeAgentDelete     = "agent.delete"
	TypeAgentResult     = "agent.result"

	// Chat history
	TypeChatHistory       = "chat.history"
	TypeChatHistoryResult = "chat.history.result"

	// Tool status
	TypeChatToolStatus = "chat.tool_status"

	// Memory search
	TypeMemorySearch       = "memory.search"
	TypeMemorySearchResult = "memory.search.result"

	// Group chat
	TypeGroupList          = "group.list"
	TypeGroupListResult    = "group.list.result"
	TypeGroupMessages      = "group.messages"
	TypeGroupMessagesResult = "group.messages.result"
	TypeGroupSend          = "group.send"
	TypeGroupMessage       = "group.message"

	// System monitoring
	TypeSystemStatus       = "system.status"
	TypeSystemStatusResult = "system.status.result"
)

// Roles
const (
	RolePhone = "phone"
	RoleAgent = "agent"
)

// Error codes
const (
	ErrUnauthorized         = "UNAUTHORIZED"
	ErrPeerOffline          = "PEER_OFFLINE"
	ErrRateLimited          = "RATE_LIMITED"
	ErrDailyQuotaExceeded   = "DAILY_QUOTA_EXCEEDED"
	ErrMonthlyQuotaExceeded = "MONTHLY_QUOTA_EXCEEDED"
	ErrMessageTooLarge      = "MESSAGE_TOO_LARGE"
	ErrInvalidMessage       = "INVALID_MESSAGE"
)

// Attachment types
const (
	AttachmentImage = "image"
)

type Attachment struct {
	Type       string `json:"type"`
	MimeType   string `json:"mime_type"`
	ContentB64 string `json:"content_b64"`
}

type ChatSendPayload struct {
	Text         string                 `json:"text"`
	Attachments  []Attachment           `json:"attachments,omitempty"`
	Model        string                 `json:"model,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
	Conversation []ConversationMsg      `json:"conversation,omitempty"`
}

type ConversationMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatStreamPayload struct {
	Delta string `json:"delta"`
	Seq   int    `json:"seq"`
}

type ChatDonePayload struct {
	FullText string     `json:"full_text"`
	Usage    *UsageInfo `json:"usage,omitempty"`
}

type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type AuthPayload struct {
	Token string `json:"token"`
	Role  string `json:"role"`
}

type AuthOkPayload struct {
	Paired     bool  `json:"paired"`
	DailyQuota int64 `json:"daily_quota_bytes,omitempty"`
	DailyUsed  int64 `json:"daily_used_bytes,omitempty"`
}

type StatusPayload struct {
	Peer string `json:"peer"` // "online" | "offline"
}

type ErrorPayload struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	RetryAfterMs int64  `json:"retry_after_ms,omitempty"`
}
