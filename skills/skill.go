package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Skill represents a skill document that can be used for task orchestration.
type Skill struct {
	// Name is the name of the skill, typically derived from the filename
	Name string

	// Content is the raw markdown content of the skill document
	Content string

	// Description is extracted from the first paragraph or header
	Description string

	// Steps contains the parsed steps/instructions from the markdown
	Steps []string

	// UsageTips contains usage suggestions/guidelines for when to use this skill
	UsageTips []string
}

// Load loads all markdown skill files from the specified directory.
// It returns a slice of Skill objects, one for each .md file found.
//
// Example:
//
//	skillList, err := skills.Load("./skills")
//	if err != nil {
//	    log.Fatal(err)
//	}
func Load(dir string) ([]Skill, error) {
	var skills []Skill

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	// Walk through the directory and find all .md files
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .md files
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		// Read the markdown file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Parse the skill
		skill := parseSkill(path, string(content))
		skills = append(skills, skill)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return skills, nil
}

// LoadFiles loads skills from an explicit list of markdown files.
// Each path in the files slice should point to a single markdown file.
//
// Example:
//
//	skillList, err := skills.LoadFiles([]string{
//	    "./skills/example_skill.md",
//	    "./skills/another_skill.md",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
func LoadFiles(files []string) ([]Skill, error) {
	var result []Skill

	for _, path := range files {
		if strings.TrimSpace(path) == "" {
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("failed to access file %s: %w", path, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("%s is a directory, expected a single file", path)
		}

		// Only accept markdown files by convention
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil, fmt.Errorf("%s is not a markdown (.md) file", path)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", path, err)
		}

		skill := parseSkill(path, string(content))
		result = append(result, skill)
	}

	return result, nil
}

// parseSkill parses a markdown file and extracts skill information.
func parseSkill(filePath, content string) Skill {
	// Extract name from filename (without extension)
	name := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	skill := Skill{
		Name:    name,
		Content: content,
	}

	// Extract description from first paragraph or first header
	lines := strings.Split(content, "\n")
	var descriptionLines []string
	var inCodeBlock bool

	for i, line := range lines {
		// Track code blocks
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		// Skip empty lines at the start
		if len(descriptionLines) == 0 && strings.TrimSpace(line) == "" {
			continue
		}

		// Stop at first header (after we have some content) or second paragraph
		if strings.HasPrefix(line, "#") && len(descriptionLines) > 0 {
			break
		}

		// Collect description lines
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			descriptionLines = append(descriptionLines, trimmed)
			// Stop after first paragraph if we hit an empty line
			if i > 0 && strings.TrimSpace(lines[i-1]) != "" && trimmed != "" {
				// Continue collecting until we hit an empty line or header
			}
		} else if len(descriptionLines) > 0 {
			// Hit empty line after description, stop
			break
		}
	}

	skill.Description = strings.Join(descriptionLines, " ")

	// Extract steps from markdown (look for numbered lists, bullet points, or ## Steps section)
	skill.Steps = extractSteps(content)

	// Extract usage tips from markdown (look for ## 使用建议 or ## Usage section)
	skill.UsageTips = extractUsageTips(content)

	return skill
}

// extractSteps extracts steps/instructions from markdown content.
// It only collects steps from the "## Steps" or "## 步骤" section.
func extractSteps(content string) []string {
	var steps []string
	lines := strings.Split(content, "\n")
	var inStepsSection bool
	var inCodeBlock bool

	for _, line := range lines {
		// Track code blocks
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Check if we're entering a steps section (support both English and Chinese)
		if strings.HasPrefix(trimmed, "##") {
			lowerTrimmed := strings.ToLower(trimmed)
			if strings.Contains(lowerTrimmed, "step") ||
				strings.Contains(lowerTrimmed, "instruction") ||
				strings.Contains(lowerTrimmed, "process") ||
				strings.Contains(lowerTrimmed, "步骤") {
				inStepsSection = true
				continue
			}
			// If we hit another section header while in steps section, stop
			if inStepsSection {
				break
			}
		}

		// Only collect steps when we're in the steps section
		if inStepsSection {
			// Collect numbered or bulleted list items
			if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") ||
				strings.HasPrefix(trimmed, "1.") || strings.HasPrefix(trimmed, "2.") ||
				strings.HasPrefix(trimmed, "3.") || strings.HasPrefix(trimmed, "4.") ||
				strings.HasPrefix(trimmed, "5.") || strings.HasPrefix(trimmed, "6.") ||
				strings.HasPrefix(trimmed, "7.") || strings.HasPrefix(trimmed, "8.") ||
				strings.HasPrefix(trimmed, "9.") || strings.HasPrefix(trimmed, "1、") ||
				strings.HasPrefix(trimmed, "2、") || strings.HasPrefix(trimmed, "3、") ||
				strings.HasPrefix(trimmed, "4、") || strings.HasPrefix(trimmed, "5、") ||
				strings.HasPrefix(trimmed, "6、") || strings.HasPrefix(trimmed, "7、") ||
				strings.HasPrefix(trimmed, "8、") || strings.HasPrefix(trimmed, "9、") {
				// Remove list markers
				step := strings.TrimSpace(trimmed)
				step = strings.TrimPrefix(step, "-")
				step = strings.TrimPrefix(step, "*")
				for i := 1; i <= 9; i++ {
					step = strings.TrimPrefix(step, fmt.Sprintf("%d.", i))
					step = strings.TrimPrefix(step, fmt.Sprintf("%d、", i))
				}
				step = strings.TrimSpace(step)
				if step != "" {
					steps = append(steps, step)
				}
			}
		}
	}

	return steps
}

// extractUsageTips extracts usage tips/suggestions from markdown content.
// It looks for "## 使用建议" or "## Usage" sections.
func extractUsageTips(content string) []string {
	var tips []string
	lines := strings.Split(content, "\n")
	var inUsageSection bool
	var inCodeBlock bool

	for _, line := range lines {
		// Track code blocks
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Check if we're entering a usage tips section (support both English and Chinese)
		if strings.HasPrefix(trimmed, "##") {
			lowerTrimmed := strings.ToLower(trimmed)
			if strings.Contains(lowerTrimmed, "usage") ||
				strings.Contains(lowerTrimmed, "使用建议") ||
				strings.Contains(lowerTrimmed, "使用") {
				inUsageSection = true
				continue
			}
			// If we hit another section header while in usage section, stop
			if inUsageSection {
				break
			}
		}

		// Only collect tips when we're in the usage section
		if inUsageSection {
			// Collect bulleted list items or plain text lines
			if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") {
				tip := strings.TrimSpace(trimmed)
				tip = strings.TrimPrefix(tip, "-")
				tip = strings.TrimPrefix(tip, "*")
				tip = strings.TrimSpace(tip)
				if tip != "" {
					tips = append(tips, tip)
				}
			} else if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				// Also collect non-empty lines that aren't headers
				tips = append(tips, trimmed)
			}
		}
	}

	return tips
}

// String returns a formatted string representation of the skill.
func (s Skill) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Skill: %s\n", s.Name))
	if s.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", s.Description))
	}
	if len(s.Steps) > 0 {
		sb.WriteString("Steps:\n")
		for i, step := range s.Steps {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
		}
	}
	if len(s.UsageTips) > 0 {
		sb.WriteString("Usage Tips:\n")
		for _, tip := range s.UsageTips {
			sb.WriteString(fmt.Sprintf("  - %s\n", tip))
		}
	}
	return sb.String()
}
