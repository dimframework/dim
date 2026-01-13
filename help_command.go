package dim

import (
	"fmt"
	"sort"
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
	fmt.Println("Dim Framework CLI")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println()

	// Find longest command name for alignment
	maxLen := 0
	for name := range c.console.commands {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	// Display commands in consistent order
	commandOrder := []string{"serve", "migrate", "migrate:rollback", "migrate:list", "route:list", "help"}

	for _, name := range commandOrder {
		if cmd, exists := c.console.commands[name]; exists {
			fmt.Printf("  %-*s  %s\n", maxLen, name, cmd.Description())
		}
	}

	// Collect and sort custom commands not in the predefined order
	var customCommands []string
	for name := range c.console.commands {
		// Check if already displayed
		inOrder := false
		for _, orderedName := range commandOrder {
			if name == orderedName {
				inOrder = true
				break
			}
		}
		if !inOrder {
			customCommands = append(customCommands, name)
		}
	}

	// Sort custom commands alphabetically
	sort.Strings(customCommands)

	// Display sorted custom commands
	for _, name := range customCommands {
		cmd := c.console.commands[name]
		fmt.Printf("  %-*s  %s\n", maxLen, name, cmd.Description())
	}

	fmt.Println()
	fmt.Println("Use '<command> -h' for more information about a command.")

	return nil
}
