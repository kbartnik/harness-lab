// Package loop contains the agent loop.
package loop

import (
	"fmt"

	"github.com/kbartnik/harness-lab/internal/core"
	"github.com/kbartnik/harness-lab/internal/provider"
	"github.com/kbartnik/harness-lab/internal/tool"
)

const maxToolIterations = 3

type Send func(messages []core.Message, tools []tool.Tool) (provider.Result, error)

func findTool(tools []tool.Tool, name string) tool.Tool {
	for _, t := range tools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}

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
				return nil, err
			}

			messages = append(messages, core.Message{Role: "tool", ToolCallID: call.ID, Text: tResult.Output})
		}
	}
}
