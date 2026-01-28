package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nov11/nacos-cli/internal/client"
	"github.com/nov11/nacos-cli/internal/listener"
	"github.com/nov11/nacos-cli/internal/skill"
)

// SkillSyncer handles skill synchronization with Nacos
type SkillSyncer struct {
	client       *client.NacosClient
	skillService *skill.SkillService
	outputDir    string
}

// NewSkillSyncer creates a new skill syncer
func NewSkillSyncer(nacosClient *client.NacosClient, outputDir string) *SkillSyncer {
	if outputDir == "" {
		homeDir, _ := os.UserHomeDir()
		outputDir = filepath.Join(homeDir, ".skills")
	}

	return &SkillSyncer{
		client:       nacosClient,
		skillService: skill.NewSkillService(nacosClient),
		outputDir:    outputDir,
	}
}

// StartSync starts synchronizing a skill with Nacos
func (s *SkillSyncer) StartSync(skillName string, stopCh <-chan struct{}) error {
	return s.StartSyncMultiple([]string{skillName}, stopCh)
}

// StartSyncMultiple starts synchronizing multiple skills with Nacos
func (s *SkillSyncer) StartSyncMultiple(skillNames []string, stopCh <-chan struct{}) error {
	if len(skillNames) == 0 {
		return fmt.Errorf("no skills specified")
	}

	fmt.Printf("Initializing synchronization for %d skill(s)...\n\n", len(skillNames))

	// Fetch initial configurations and build listening items
	var items []listener.ConfigItem
	var existingCount int
	for _, skillName := range skillNames {
		group := fmt.Sprintf("skill_%s", skillName)

		fmt.Printf("[%s] Fetching initial configuration\n", skillName)
		content, err := s.client.GetConfig("skill.json", group)
		if err != nil {
			// Check if it's a 404 error
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not exist") {
				fmt.Printf("  - Skill not found in Nacos, will monitor for creation\n")
				// Remove local copy if exists
				skillPath := filepath.Join(s.outputDir, skillName)
				if _, statErr := os.Stat(skillPath); statErr == nil {
					fmt.Printf("  - Removing stale local copy...\n")
					if rmErr := os.RemoveAll(skillPath); rmErr != nil {
						fmt.Printf("  - Failed to remove: %v\n", rmErr)
					} else {
						fmt.Printf("  - Local copy removed\n")
					}
				}
				// Still add to listening items with empty MD5 to monitor for creation
				items = append(items, listener.ConfigItem{
					DataID: "skill.json",
					Group:  group,
					Tenant: s.client.Namespace,
					MD5:    "", // Empty MD5 for non-existent config
				})
				continue
			}
			fmt.Printf("  - Failed to fetch: %v\n", err)
			continue
		}
		if content == "" {
			fmt.Printf("  - Skill not found\n")
			continue
		}

		// Calculate initial MD5
		initialMD5 := calculateMD5(content)
		fmt.Printf("  - MD5: %s\n", initialMD5[:8])

		// Download skill initially
		fmt.Printf("  - Downloading to %s...\n", s.outputDir)
		if err := s.skillService.GetSkill(skillName, s.outputDir); err != nil {
			fmt.Printf("  - Failed to download: %v\n", err)
			continue
		}
		fmt.Printf("  - Downloaded successfully\n\n")
		existingCount++

		// Add to listening items
		items = append(items, listener.ConfigItem{
			DataID: "skill.json",
			Group:  group,
			Tenant: s.client.Namespace,
			MD5:    initialMD5,
		})
	}

	if len(items) == 0 {
		return fmt.Errorf("no skills specified for monitoring")
	}

	if existingCount > 0 {
		fmt.Printf("Successfully initialized %d existing skill(s)\n\n", existingCount)
	} else {
		fmt.Printf("No existing skills found, monitoring for creation\n\n")
	}

	// Create config listener
	configListener := listener.NewConfigListener(s.client.ServerAddr, s.client.Username, s.client.Password)

	// Define change handler
	handler := func(dataID, grp, tenant string) error {
		// Extract skill name from group (skill_xxx)
		skillName := ""
		if strings.HasPrefix(grp, "skill_") {
			skillName = strings.TrimPrefix(grp, "skill_")
		}

		fmt.Printf("\n[%s] Configuration changed\n", skillName)
		if skillName != "" {
			// First check if the skill exists
			content, err := s.client.GetConfig(dataID, grp)
			if err != nil || content == "" {
				// Check if it's a 404 error (skill was deleted)
				if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not exist") {
					fmt.Printf("  - Skill deleted from Nacos, removing local copy...\n")
					skillPath := filepath.Join(s.outputDir, skillName)
					if err := os.RemoveAll(skillPath); err != nil {
						fmt.Printf("  - Failed to remove local skill: %v\n", err)
					} else {
						fmt.Printf("  - Local skill removed: %s\n", skillPath)
					}
					return nil // Don't treat this as an error
				}
				return fmt.Errorf("failed to fetch config: %w", err)
			}

			// Skill exists, sync it
			fmt.Printf("  - Syncing skill...\n")
			if err := s.skillService.GetSkill(skillName, s.outputDir); err != nil {
				return fmt.Errorf("failed to sync skill: %w", err)
			}

			skillPath := filepath.Join(s.outputDir, skillName)
			fmt.Printf("  - Synced successfully to %s\n", skillPath)
		}
		return nil
	}

	// Start listening
	fmt.Printf("Listening for changes to %d skill(s):\n", len(items))
	for _, item := range items {
		skillName := strings.TrimPrefix(item.Group, "skill_")
		fmt.Printf("  - %s\n", skillName)
	}
	fmt.Printf("\nPress Ctrl+C to stop\n\n")

	return configListener.StartListening(items, handler, stopCh)
}

// calculateMD5 is a helper function (exported from listener package)
func calculateMD5(content string) string {
	return listener.CalculateMD5(content)
}
