package help

import (
	"fmt"
	"strings"
)

// CommandHelp defines the help information for a command
type CommandHelp struct {
	Command     string
	Description string
	Parameters  []string
	Examples    []string
}

// All command help definitions
var (
	SkillList = CommandHelp{
		Command:     "skill-list",
		Description: "List all skills from Nacos configuration center.",
		Parameters: []string{
			"--name string   Filter by skill name (supports wildcard *)",
			"--page int      Page number (default: 1)",
			"--size int      Page size (default: 20)",
		},
		Examples: []string{
			"# List all skills",
			"skill-list",
			"",
			"# Search by name",
			"skill-list --name \"creator\"",
			"",
			"# With pagination",
			"skill-list --page 2 --size 10",
		},
	}

	SkillGet = CommandHelp{
		Command:     "skill-get",
		Description: "Download a skill from Nacos to local ~/.skills directory.",
		Parameters: []string{
			"skillName       Required. The name of the skill to download",
		},
		Examples: []string{
			"# Download a skill",
			"skill-get skill-creator",
			"",
			"# Download will create ~/.skills/skill-creator/ with:",
			"#   - SKILL.md (documentation)",
			"#   - scripts/ (script files)",
			"#   - references/ (reference documents)",
		},
	}

	SkillUpload = CommandHelp{
		Command:     "skill-upload",
		Description: "Upload a skill directory to Nacos as a ZIP file.",
		Parameters: []string{
			"skillPath       Required. Path to the skill directory",
			"--all           Upload all skills in the specified directory",
		},
		Examples: []string{
			"# Upload a single skill",
			"skill-upload ./my-skill",
			"",
			"# Upload all skills in a directory",
			"skill-upload --all ./skills-folder",
			"",
			"Note:",
			"  - Skill directory must contain SKILL.md",
			"  - Skill names: letters, underscores (_), hyphens (-) only",
		},
	}

	ConfigList = CommandHelp{
		Command:     "config-list",
		Description: "List all configurations from Nacos configuration center.",
		Parameters: []string{
			"--data-id string   Filter by data ID (supports wildcard *)",
			"--group string     Filter by group (supports wildcard *)",
			"--page int         Page number (default: 1)",
			"--size int         Page size (default: 20)",
		},
		Examples: []string{
			"# List all configurations",
			"config-list",
			"",
			"# Filter by data ID",
			"config-list --data-id resource*",
			"",
			"# Filter by group",
			"config-list --group skill_*",
			"",
			"# Combine filters with pagination",
			"config-list --data-id *config* --group DEFAULT_GROUP --page 1 --size 50",
		},
	}

	ConfigGet = CommandHelp{
		Command:     "config-get",
		Description: "Get a specific configuration from Nacos.",
		Parameters: []string{
			"dataId          Required. Configuration data ID",
			"group           Required. Configuration group name",
		},
		Examples: []string{
			"# Get a configuration",
			"config-get application.yaml DEFAULT_GROUP",
			"",
			"# Get a skill configuration",
			"config-get skill.json skill_skill-creator",
		},
	}

	ConfigSet = CommandHelp{
		Command:     "config-set",
		Description: "Publish a configuration to Nacos (create or update).",
		Parameters: []string{
			"dataId          Required. Configuration data ID",
			"group           Required. Configuration group name",
			"--file, -f      Path to config file (default: read from stdin)",
		},
		Examples: []string{
			"# Publish from file",
			"config-set application.yaml DEFAULT_GROUP --file ./application.yaml",
			"",
			"# Publish from stdin",
			" echo 'key: value' | nacos-cli config-set app.yaml DEFAULT_GROUP",
			"",
			"# Publish JSON config",
			"config-set skill.json skill_my-skill -f ./skill.json",
		},
	}

	SkillSync = CommandHelp{
		Command:     "skill-sync",
		Description: "Synchronize skills with Nacos (real-time updates).",
		Parameters: []string{
			"skillName...    Optional. One or more skill names to synchronize",
			"--all           Synchronize all skills",
		},
		Examples: []string{
			"# Sync a single skill",
			"skill-sync skill-creator",
			"",
			"# Sync multiple skills",
			"skill-sync skill-creator skill-analyzer skill-formatter",
			"",
			"# Sync all skills",
			"skill-sync --all",
			"",
			"# Skills will be downloaded to ~/.skills/",
			"# Press Ctrl+C to stop synchronization",
		},
	}
)

// FormatForCLI formats help content for CLI mode (Cobra Long description)
func (h *CommandHelp) FormatForCLI(cliPrefix string) string {
	result := h.Description + "\n\nParameters:\n"
	for _, param := range h.Parameters {
		result += "  " + param + "\n"
	}
	result += "\nExamples:\n"
	for _, example := range h.Examples {
		if example == "" {
			result += "\n"
		} else {
			// Replace command name with CLI prefix
			if example[0] != '#' && example[0] != ' ' && example != "Note:" {
				result += "  " + cliPrefix + " " + example + "\n"
			} else {
				result += "  " + example + "\n"
			}
		}
	}
	return result
}

// FormatForTerminal formats help content for terminal mode with colors
func (h *CommandHelp) FormatForTerminal() {
	fmt.Printf("\033[1;36mCommand: %s\033[0m\n", h.Command)
	fmt.Printf("\n%s\n\n", h.Description)
	fmt.Println("\033[33mParameters:\033[0m")
	for _, param := range h.Parameters {
		fmt.Printf("  %s\n", param)
	}
	fmt.Println()
	fmt.Println("\033[33mExamples:\033[0m")
	for _, example := range h.Examples {
		if example == "" {
			fmt.Println()
		} else if strings.HasPrefix(example, "Note:") || strings.HasPrefix(example, "  -") {
			fmt.Printf("\033[33m%s\033[0m\n", example)
		} else {
			fmt.Printf("  %s\n", example)
		}
	}
}
