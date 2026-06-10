package bus

type ToolName string

const (
	ToolBattery          ToolName = "battery"
	ToolLocation         ToolName = "location"
	ToolDevice           ToolName = "device"
	ToolWifi             ToolName = "wifi"
	ToolClipboardGet     ToolName = "clipboard_get"
	ToolRead             ToolName = "read"
	ToolShell            ToolName = "shell"
	ToolCamera           ToolName = "camera"
	ToolTorchOn          ToolName = "torch_on"
	ToolTorchOff         ToolName = "torch_off"
	ToolToast            ToolName = "toast"
	ToolTTS              ToolName = "tts"
	ToolVibrate          ToolName = "vibrate"
	ToolNotification     ToolName = "notification"
	ToolIntentView       ToolName = "intent_view"
	ToolIntentDial       ToolName = "intent_dial"
	ToolIntentSettings   ToolName = "intent_settings"
	ToolIntentAlarm      ToolName = "intent_alarm"
	ToolShizuku          ToolName = "shizuku"
	ToolNexoUIDump       ToolName = "nexo_ui_dump"
	ToolNexoScreencap    ToolName = "nexo_screencap"
	ToolNexoInputTap     ToolName = "nexo_input_tap"
	ToolNexoInputText    ToolName = "nexo_input_text"
	ToolNexoInputKeyevent ToolName = "nexo_input_keyevent"
	ToolNexoIntent       ToolName = "nexo_intent"
	ToolNexoBsh          ToolName = "nexo_bsh"
	ToolNexoBshExpr      ToolName = "nexo_bsh_expr"
	ToolNexoContentQuery ToolName = "nexo_content_query"
)

type ToolCapability struct {
	Name         ToolName    `json:"name"`
	Keywords     []string    `json:"keywords"`
	Params       []ToolParam `json:"params"`
	Description  string      `json:"description"`
	Category     string      `json:"category"`
	RequiresAuth bool        `json:"requires_auth"`
}

type ToolParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

type ToolExecResult struct {
	ToolName  ToolName `json:"tool_name"`
	Success   bool     `json:"success"`
	Data      string   `json:"data"`
	Error     string   `json:"error,omitempty"`
	Latency   int64    `json:"latency_ms"`
	Timestamp int64    `json:"timestamp"`
}
