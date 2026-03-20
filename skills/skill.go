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

func LoadContents(contents []string) ([]Skill, error) {
	var result []Skill

	for _, content := range contents {
		skill := parseSkill("", content)
		result = append(result, skill)
	}

	return result, nil
}

// parseSkill parses a markdown file and extracts skill information.
func parseSkill(filePath, content string) Skill {
	// Extract name from filename (without extension)
	name := ""
	if filePath != "" {
		name = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}

	lines := strings.Split(content, "\n")

	// if name is empty, use the first line of the content as the name
	if name == "" {
		name = strings.Replace(lines[0], "#", "", 1)
		name = strings.TrimSpace(name)
	}

	skill := Skill{
		Name:    name,
		Content: content,
	}

	// Extract description from first paragraph (before any ## sections)
	var descriptionLines []string
	var inCodeBlock bool

	for index, line := range lines {

		if index == 0 {
			continue
		}

		// Track code blocks
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		// Skip the first-level header (title)
		if strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "##") {
			break
		}

		// Skip empty lines at the start
		if len(descriptionLines) == 0 && trimmed == "" {
			continue
		}

		// Collect description lines (non-empty lines)
		if trimmed != "" {
			descriptionLines = append(descriptionLines, trimmed)
		}
	}

	skill.Description = strings.Join(descriptionLines, " | ")

	return skill
}
