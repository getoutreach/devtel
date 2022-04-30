package devtel

import "strings"

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
