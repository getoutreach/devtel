package devspace

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// hookCombinations is a list of all interesting start, end event combinations.
// it's not an exhaustive list of hooks, but it has the ones that are in pairs (except plugin ones).
var hookCombinations = [][]string{
	{"before:build", "after:build", "error:build"},
	{"before:deploy", "after:deploy"},
	{"before:deploy", "after:deploy", "error:deploy", "skip:deploy"},
	{"before:render", "after:render"},
	{"before:render", "after:render", "error:render"},
	{"before:purge", "after:purge"},
	{"before:purge", "after:purge", "error:purge"},
	{"before:resolveDependency", "after:resolveDependency", "error:resolveDependency"},
	{"before:buildDependency", "after:buildDependency", "error:buildDependency"},
	{"before:deployDependency", "after:deployDependency", "error:deployDependency"},
	{"before:renderDependency", "after:renderDependency", "error:renderDependency"},
	{"before:purgeDependency", "after:purgeDependency", "error:purgeDependency"},
	{"before:configLoad", "after:configLoad", "error:configLoad"},
	{"start:sync", "stop:sync", "error:sync", "restart:sync"},
	{"before:initialSync", "after:initialSync", "error:initialSync"},
	{"start:portForwarding", "error:portForwarding", "stop:portForwarding"},
	{"start:reversePortForwarding", "error:reversePortForwarding", "stop:reversePortForwarding"},
	{"before:createPullSecrets", "after:createPullSecrets", "error:createPullSecrets"},
	{"devCommand:before:sync", "devCommand:after:sync"},
	{"devCommand:before:portForwarding", "devCommand:after:portForwarding"},
	{"devCommand:before:replacePods", "devCommand:after:replacePods"},
	{"devCommand:before:runPipeline", "devCommand:after:runPipeline"},
	{"devCommand:before:deployDependencies", "devCommand:after:deployDependencies"},
	{"devCommand:before:build", "devCommand:after:build"},
	{"devCommand:before:deploy", "devCommand:after:deploy"},
	{"devCommand:before:execute", "devCommand:after:execute", "devCommand:interrupt", "devCommand:error"},
	{"deployCommand:before:execute", "deployCommand:after:execute", "deployCommand:error", "deployCommand:interrupt"},
	{"purgeCommand:before:execute", "purgeCommand:after:execute", "purgeCommand:error", "purgeCommand:interrupt"},
	{"buildCommand:before:execute", "buildCommand:after:execute", "buildCommand:error", "buildCommand:interrupt"},
	{"command:before:execute", "command:after:execute", "command:error"},
}

// getBeforeHook returns the start event for the given event.
// 1. If the event is a start event, it returns "".
// 2. If the event is an end event (after:deploy, or error:deploy for example), it returns the start event (before:deploy).
// 3. If the event has a sub-event specified, (after:deploy:app), it returns the start event with sub-event specified (before:deploy:app).
func getBeforeHook(hook string) string {
	for _, combination := range hookCombinations {
		for i, h := range combination {
			// The first hook in the combination is the start hook
			if i == 0 {
				continue
			}
			if strings.HasPrefix(hook, h) {
				return strings.Replace(hook, h, combination[0], 1)
			}
		}
	}

	return ""
}

type Command struct {
	Name  string   `json:"name"`
	Line  string   `json:"line"`
	Flags []string `json:"flags,omitempty"`
	Args  []string `json:"args,omitempty"`
}

type Event struct {
	Name string `json:"event,omitempty"`

	Hook string `json:"hook,omitempty"`

	ExecutionID string `json:"execution_id,omitempty"`

	Error  string `json:"error,omitempty"`
	Status string `json:"status,omitempty"`

	Command *Command `json:"command,omitempty"`

	Timestamp int64 `json:"timestamp"`
	Duration  int64 `json:"duration_ms,omitempty"`
}

func (e *Event) Key() string {
	if e.ExecutionID == "" {
		return e.Hook
	}

	return fmt.Sprintf("%s_%s", e.ExecutionID, e.Hook)
}

func (e *Event) MarshalRecord(addField func(name string, value interface{})) {
	addField("event", e.Name)
	addField("hook", e.Hook)
	addField("execution_id", e.ExecutionID)
	if e.Error != "" {
		addField("error", e.Error)
	}
	addField("status", e.Status)
	addField("timestamp", e.Timestamp)
	if e.Duration != 0 {
		addField("duration_ms", e.Duration)
	}

	if e.Command != nil {
		addField("command.name", e.Command.Name)
		addField("command.line", e.Command.Line)
		addField("command.flags", e.Command.Flags)
		addField("command.args", e.Command.Args)
	}
}

func (e *Event) UnmarshalRecord(data map[string]interface{}) error {
	if val, ok := data["event"]; ok {
		e.Name = val.(string)
	}
	if val, ok := data["hook"]; ok {
		e.Hook = val.(string)
	}
	if val, ok := data["execution_id"]; ok {
		e.ExecutionID = val.(string)
	}

	if val, ok := data["error"]; ok {
		e.Error = val.(string)
	}
	if val, ok := data["status"]; ok {
		e.Status = val.(string)
	}

	if val, ok := data["command"]; ok {
		if cmd, ok := val.(map[string]interface{}); ok {
			e.Command = &Command{}
			if val, ok := cmd["name"]; ok {
				e.Command.Name = val.(string)
			}
			if val, ok := cmd["line"]; ok {
				e.Command.Line = val.(string)
			}
			if val, ok := cmd["flags"]; ok {
				switch val := val.(type) {
				case []interface{}:
					for _, v := range val {
						e.Command.Flags = append(e.Command.Flags, v.(string))
					}
				case []string:
					e.Command.Flags = val
				}
			}
			if val, ok := cmd["args"]; ok {
				switch val := val.(type) {
				case []interface{}:
					for _, v := range val {
						e.Command.Args = append(e.Command.Args, v.(string))
					}
				case []string:
					e.Command.Args = val
				}
			}
		}
	}

	if val, ok := data["timestamp"]; ok {
		switch t := val.(type) {
		case float64:
			e.Timestamp = int64(t)
		case int64:
			e.Timestamp = t
		}
	}
	if val, ok := data["duration_ms"]; ok {
		switch t := val.(type) {
		case float64:
			e.Duration = int64(t)
		case int64:
			e.Duration = t
		}
	}

	return nil
}

func EventFromEnv() *Event {
	var flags, args []string

	//nolint:errcheck // Why: There's not much we can do about this. We still want the rest.
	json.Unmarshal([]byte(os.Getenv("DEVSPACE_PLUGIN_COMMAND_FLAGS")), &flags)

	//nolint:errcheck // Why: There's not much we can do about this. We still want the rest.
	json.Unmarshal([]byte(os.Getenv("DEVSPACE_PLUGIN_COMMAND_ARGS")), &args)

	errMsg := os.Getenv("DEVSPACE_PLUGIN_ERROR")
	status := "info"
	if errMsg != "" {
		status = "error"
	}

	return &Event{
		Name:        "devspace_hook",
		Hook:        os.Getenv("DEVSPACE_PLUGIN_EVENT"),
		ExecutionID: os.Getenv("DEVSPACE_PLUGIN_EXECUTION_ID"),
		Error:       errMsg,
		Status:      status,

		Command: &Command{
			Name:  os.Getenv("DEVSPACE_PLUGIN_COMMAND"),
			Line:  os.Getenv("DEVSPACE_PLUGIN_COMMAND_LINE"),
			Flags: flags,
			Args:  args,
		},

		Timestamp: time.Now().UnixMilli(),
	}
}
