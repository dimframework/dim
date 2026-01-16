package dim

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// HelpCommand menampilkan daftar semua command yang tersedia.
type HelpCommand struct {
	console *Console
}

func (c *HelpCommand) Name() string {
	return "help"
}

func (c *HelpCommand) Description() string {
	return "Show available commands"
}

func (c *HelpCommand) Execute(ctx *CommandContext) error {
	// Use injected output writer (for testing) or default to stdout
	out := ctx.Out
	if out == nil {
		out = os.Stdout
	}

	fmt.Fprintln(out, "Dim Framework CLI")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Available commands:")
	fmt.Fprintln(out)

	// 1. Collect all command names
	var names []string
	for name := range c.console.commands {
		names = append(names, name)
	}
	sort.Strings(names)

	// 2. Identify namespaces
	namespaces := make(map[string]bool)
	for _, name := range names {
		if strings.Contains(name, ":") {
			parts := strings.SplitN(name, ":", 2)
			namespaces[parts[0]] = true
		}
	}

	// 3. Group commands
	rootCommands := []string{}
	groupedCommands := make(map[string][]string)

	for _, name := range names {
		// Check if it belongs to a namespace
		var processed bool

		// Case 1: has prefix
		if strings.Contains(name, ":") {
			parts := strings.SplitN(name, ":", 2)
			ns := parts[0]
			groupedCommands[ns] = append(groupedCommands[ns], name)
			processed = true
		} else {
			// Case 2: is a namespace root (e.g. "migrate")
			if namespaces[name] {
				groupedCommands[name] = append(groupedCommands[name], name)
				processed = true
			}
		}

		// Case 3: Root command
		if !processed {
			rootCommands = append(rootCommands, name)
		}
	}

	// Calculate padding for alignment
	// We want descriptions to be aligned globally.
	// Root commands indent: 2 spaces
	// Group commands indent: 4 spaces (2 + 2)
	maxVisualLen := 0
	for _, name := range rootCommands {
		if len(name) > maxVisualLen {
			maxVisualLen = len(name)
		}
	}
	for _, cmds := range groupedCommands {
		for _, name := range cmds {
			// +2 for extra indentation
			if len(name)+2 > maxVisualLen {
				maxVisualLen = len(name) + 2
			}
		}
	}

	// Print Root Commands
	for _, name := range rootCommands {
		cmd := c.console.commands[name]
		fmt.Fprintf(out, "  %-*s  %s\n", maxVisualLen, name, cmd.Description())
	}

	// Sort and Print Groups
	var sortedGroups []string
	for ns := range groupedCommands {
		sortedGroups = append(sortedGroups, ns)
	}
	sort.Strings(sortedGroups)

	for _, ns := range sortedGroups {
		cmds := groupedCommands[ns]
		fmt.Fprintf(out, "  %s\n", ns)

		// Sort commands within group? They are already sorted because 'names' was sorted,
		// but let's be sure.
		// 'names' is sorted alphabetically. Populating 'groupedCommands' by iterating 'names' maintains order.
		// So cmds is sorted.

		for _, name := range cmds {
			cmd := c.console.commands[name]
			// Indent 4 spaces. Width for padding is maxVisualLen - 2.
			fmt.Fprintf(out, "    %-*s  %s\n", maxVisualLen-2, name, cmd.Description())
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Use '<command> -h' for more information about a command.")

	return nil
}
