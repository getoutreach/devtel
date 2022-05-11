// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains
// - devspace hook combinations
// - Event marshalling and unmarshalling.
// - env variables scraping for devspace details.

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

// Command contains the details about devspace command that triggered the event.
type Command struct {
	Name  string   `json:"name"`
	Line  string   `json:"line"`
	Flags []string `json:"flags,omitempty"`
	Args  []string `json:"args,omitempty"`
}

// Devenv contains the details about devenv environment variables provided with the event.
type Devenv struct {
	Bin                    string `json:"bin,omitempty"`
	Version                string `json:"version,omitempty"`
	KindBin                string `json:"kind_bin,omitempty"`
	DevspaceBin            string `json:"devspace_bin,omitempty"`
	Type                   string `json:"type,omitempty"`
	DevDeploymentProfile   string `json:"dev_deployment_profile,omitempty"`
	DeployVersion          string `json:"deploy_version,omitempty"`
	DeployImageSource      string `json:"deploy_image_source,omitempty"`
	DeployImageRegistry    string `json:"deploy_image_registry,omitempty"`
	DeployDevImageRegistry string `json:"deploy_dev_image_registry,omitempty"`
	DeployBoxImageRegistry string `json:"deploy_box_image_registry,omitempty"`
	DeployAppname          string `json:"deploy_appname,omitempty"`

	DeployUseDevspace     bool `json:"deploy_use_devspace,omitempty"`
	DevSkipPortforwarding bool `json:"dev_skip_portforwarding,omitempty"`
	DevTerminal           bool `json:"dev_terminal,omitempty"`
}

// Event contains the details about the devspace hook event.
type Event struct {
	Name string `json:"event,omitempty"`

	Hook string `json:"hook,omitempty"`

	ExecutionID string `json:"execution_id,omitempty"`

	Error  string `json:"error,omitempty"`
	Status string `json:"status,omitempty"`

	Command *Command `json:"command,omitempty"`
	Devenv  *Devenv  `json:"devenv,omitempty"`

	Timestamp    int64     `json:"timestamp"`
	TimestampTag time.Time `json:"@timestamp,omitempty"`
	Duration     int64     `json:"duration_ms,omitempty"`
}

// Key returns the key for the event index.
func (e *Event) Key() string {
	if e.ExecutionID == "" {
		return e.Hook
	}

	return fmt.Sprintf("%s_%s", e.ExecutionID, e.Hook)
}

// MarshalRecord adds the event data to the target data structure (map most likely).
func (e *Event) MarshalRecord(addField func(name string, value interface{})) {
	addField("event", e.Name)
	addField("hook", e.Hook)
	addField("execution_id", e.ExecutionID)

	if e.Error != "" {
		addField("error", e.Error)
	}
	addField("status", e.Status)

	addField("timestamp", e.Timestamp)
	if !e.TimestampTag.IsZero() {
		addField("@timestamp", e.TimestampTag)
	}
	if e.Duration != 0 {
		addField("duration_ms", e.Duration)
	}

	if e.Command != nil {
		addField("command.name", e.Command.Name)
		addField("command.line", e.Command.Line)
		if len(e.Command.Flags) > 0 {
			addField("command.flags", e.Command.Flags)
		}
		if len(e.Command.Args) > 0 {
			addField("command.args", e.Command.Args)
		}
	}

	if e.Devenv != nil {
		addField("devenv.runtime", e.Devenv.Type)

		addField("devenv.bin", e.Devenv.Bin)
		addField("devenv.version", e.Devenv.Version)
		addField("devenv.kind_bin", e.Devenv.KindBin)
		addField("devenv.devspace_bin", e.Devenv.DevspaceBin)
		addField("devenv.dev_deployment_profile", e.Devenv.DevDeploymentProfile)
		addField("devenv.deploy_version", e.Devenv.DeployVersion)
		addField("devenv.deploy_image_source", e.Devenv.DeployImageSource)
		addField("devenv.deploy_image_registry", e.Devenv.DeployImageRegistry)
		addField("devenv.deploy_dev_image_registry", e.Devenv.DeployDevImageRegistry)
		addField("devenv.deploy_box_image_registry", e.Devenv.DeployBoxImageRegistry)
		addField("devenv.deploy_appname", e.Devenv.DeployAppname)

		addField("devenv.deploy_use_devspace", e.Devenv.DeployUseDevspace)
		addField("devenv.dev_skip_portforwarding", e.Devenv.DevSkipPortforwarding)
		addField("devenv.dev_terminal", e.Devenv.DevTerminal)
	}
}

// UnmarshalRecord unmarshals the event data from the map into Event.
func (e *Event) UnmarshalRecord(data map[string]interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, e)
}

// EventFromEnv scrapes the event data from the environment variables.
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

	t := time.Now()

	command := &Command{
		Name:  os.Getenv("DEVSPACE_PLUGIN_COMMAND"),
		Line:  os.Getenv("DEVSPACE_PLUGIN_COMMAND_LINE"),
		Flags: flags,
		Args:  args,
	}

	devenv := &Devenv{
		Bin:                    os.Getenv("DEVENV_BIN"),
		Version:                os.Getenv("DEVENV_VERSION"),
		KindBin:                os.Getenv("DEVENV_KIND_BIN"),
		DevspaceBin:            os.Getenv("DEVENV_DEVSPACE_BIN"),
		Type:                   os.Getenv("DEVENV_TYPE"),
		DevDeploymentProfile:   os.Getenv("DEVENV_DEV_DEPLOYMENT_PROFILE"),
		DeployVersion:          os.Getenv("DEVENV_DEPLOY_VERSION"),
		DeployImageSource:      os.Getenv("DEVENV_DEPLOY_IMAGE_SOURCE"),
		DeployImageRegistry:    os.Getenv("DEVENV_DEPLOY_IMAGE_REGISTRY"),
		DeployDevImageRegistry: os.Getenv("DEVENV_DEPLOY_DEV_IMAGE_REGISTRY"),
		DeployBoxImageRegistry: os.Getenv("DEVENV_DEPLOY_BOX_IMAGE_REGISTRY"),
		DeployAppname:          os.Getenv("DEVENV_DEPLOY_APPNAME"),

		DeployUseDevspace:     os.Getenv("DEVENV_DEPLOY_USE_DEVSPACE") != "",
		DevTerminal:           os.Getenv("DEVENV_DEV_TERMINAL") != "",
		DevSkipPortforwarding: os.Getenv("DEVENV_DEV_SKIP_PORTFORWARDING") != "",
	}

	return &Event{
		Name:        "devspace_hook",
		Hook:        os.Getenv("DEVSPACE_PLUGIN_EVENT"),
		ExecutionID: os.Getenv("DEVSPACE_PLUGIN_EXECUTION_ID"),
		Error:       errMsg,
		Status:      status,

		Command: command,
		Devenv:  devenv,

		Timestamp:    t.UnixMilli(),
		TimestampTag: t,
	}
}
