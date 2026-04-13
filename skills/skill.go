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
	// Name from front matter (name: ...) or else the file base name without extension.
	Name string

	// Description from front matter (description: ...).
	Description string

	// Path is the absolute path to the markdown file after loading via LoadFiles, LoadDirectory, or Load.
	Path string
}

func Load(skills []Skill) ([]Skill, error) {
	var result []Skill

	for _, skill := range skills {
		if skill.Path == "" {
			continue
		}

		abs, err := filepath.Abs(skill.Path)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err != nil {
			continue
		}

		skill.Path = abs
		result = append(result, skill)
	}

	return result, nil
}

// Load loads skill markdown files from the specified directory.
// Preferred structure is one skill per subdirectory:
//
//	skills/
//	├── skill-management/
//	│   ├── SKILL.md
//
// LoadDirectory scans recursively and only loads files named SKILL.md (case-insensitive).
//
// Example:
//
//	skillList, err := skills.Load("./skills")
//	if err != nil {
//	    log.Fatal(err)
//	}
func LoadDirectory(dir string) ([]Skill, error) {
	var skills []Skill

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	// Walk through the directory and find all SKILL.md files
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if !d.IsDir() {
			return nil
		}

		filePath := filepath.Join(path, "SKILL.md")

		// if not exists, skip
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return nil
		}

		abs, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("failed to resolve path %s: %w", path, err)
		}

		// Read the markdown file
		content, err := os.ReadFile(abs)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", abs, err)
		}

		// Parse the skill
		skill := parseSkill(abs, string(content))
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

		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %s: %w", path, err)
		}

		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("failed to access file %s: %w", abs, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("%s is a directory, expected a single file", abs)
		}

		// Only accept markdown files by convention
		if !strings.HasSuffix(strings.ToLower(abs), ".md") {
			return nil, fmt.Errorf("%s is not a markdown (.md) file", abs)
		}

		content, err := os.ReadFile(abs)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", abs, err)
		}

		skill := parseSkill(abs, string(content))
		result = append(result, skill)
	}

	return result, nil
}

// parseSkill parses a markdown file and extracts skill information.
//
// If the file begins with YAML-style front matter between --- lines (name: / description:),
// those values are used. See skills/example.md.
func parseSkill(filePath, content string) Skill {
	name, desc := parseSkillFrontMatter(content)
	skill := Skill{
		Path:        filePath,
		Name:        name,
		Description: desc,
	}
	if skill.Name == "" {
		base := filepath.Base(filePath)
		skill.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return skill
}

func parseSkillFrontMatter(content string) (name, description string) {
	s := strings.ReplaceAll(content, "\r\n", "\n")
	s = strings.TrimPrefix(s, "\ufeff")
	lines := strings.Split(s, "\n")
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) || !isSkillFrontMatterDelimiter(lines[i]) {
		return "", ""
	}
	i++
	for i < len(lines) {
		line := lines[i]
		i++
		if isSkillFrontMatterDelimiter(line) {
			break
		}
		nl := strings.TrimSpace(line)
		if nl == "" {
			continue
		}
		if v, ok := strings.CutPrefix(nl, "name:"); ok {
			name = strings.TrimSpace(v)
		} else if v, ok := strings.CutPrefix(nl, "description:"); ok {
			description = strings.TrimSpace(v)
		}
	}
	return name, description
}

func isSkillFrontMatterDelimiter(line string) bool {
	return strings.TrimSpace(line) == "---"
}
