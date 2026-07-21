// Package loop implements the agent loop. Run sends the conversation to a
// model provider, executes any tool calls in the response, appends results,
// and repeats until the model stops or a tool-iteration limit is reached.
// Unknown tool names are fed back to the model as error content rather than
// aborting the loop.
package loop

import (
	"fmt"

	"github.com/kbartnik/harness-lab/internal/core"
	"github.com/kbartnik/harness-lab/internal/provider"
	"github.com/kbartnik/harness-lab/internal/tool"
)

// maxToolIterations caps the number of consecutive tool-call rounds per Run
// invocation. If the model keeps requesting tools beyond this limit, Run
// returns an error rather than looping indefinitely.
const maxToolIterations = 3

// Send is the function signature for delivering a conversation to a model
// provider and receiving its response. Abstracting it as a type lets the loop
// remain provider-agnostic and makes it straightforward to inject fakes in tests.
type Send func(messages []core.Message, tools []tool.Tool) (provider.Result, error)

// findTool returns the first tool in the slice whose Name matches, or nil if
// none is found. Callers treat nil as an unknown-tool condition.
func findTool(tools []tool.Tool, name string) tool.Tool {
	for _, t := range tools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}

// Run executes the agent loop: it sends messages to the model, executes any
// tool calls in the response, appends results to the conversation, and repeats
// until the model stops requesting tools or maxToolIterations is exceeded. The
// returned slice is the full updated conversation including the final assistant
// message. The input slice is never mutated.
func Run(send Send, tools []tool.Tool, messages []core.Message) ([]core.Message, error) {
	messages = append([]core.Message(nil), messages...)
	toolIterations := 0

	for {
		sResult, err := send(messages, tools)
		if err != nil {
			return nil, err
		}

		messages = append(messages, sResult.Message)

		if len(sResult.Message.ToolCalls) == 0 {
			return messages, nil
		}

		toolIterations++
		if toolIterations > maxToolIterations {
			return nil, fmt.Errorf("maxToolIterations of %d exceeded", maxToolIterations)
		}

		for _, call := range sResult.Message.ToolCalls {
			t := findTool(tools, call.Name)
			if t == nil {
				messages = append(messages, core.Message{Role: "tool", ToolCallID: call.ID, Text: fmt.Sprintf("unknown tool: %s", call.Name)})
				continue
			}

			tResult, err := t.Execute(call.Args)
			if err != nil {
				return nil, fmt.Errorf("tool %s: %w", call.Name, err)
			}

			messages = append(messages, core.Message{Role: "tool", ToolCallID: call.ID, Text: tResult.Output})
		}
	}
}
