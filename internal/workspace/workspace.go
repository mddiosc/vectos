package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Scope struct {
	Name          string   `json:"name"`
	WorkspaceRoot string   `json:"workspace_root,omitempty"`
	PrimaryRoot   string   `json:"primary_root"`
	Roots         []string `json:"roots"`
	WorkspaceType string   `json:"workspace_type,omitempty"`
}

func (s Scope) IsWorkspace() bool {
	return strings.TrimSpace(s.WorkspaceType) != ""
}

type NxProject struct {
	Name string `json:"name"`
	Root string `json:"root"`
}

func ResolveScope(path string, projectName string) (Scope, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Scope{}, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return Scope{}, err
	}

	startDir := absPath
	if !info.IsDir() {
		startDir = filepath.Dir(absPath)
	}

	workspaceRoot := detectNxWorkspaceRoot(startDir)
	if workspaceRoot == "" {
		projectRoot := detectProjectRoot(startDir)
		return Scope{
			Name:        filepath.Base(projectRoot),
			PrimaryRoot: projectRoot,
			Roots:       []string{projectRoot},
		}, nil
	}

	projects, err := discoverNxProjects(workspaceRoot)
	if err != nil {
		return Scope{}, err
	}
	if len(projects) == 0 {
		return Scope{}, fmt.Errorf("nx workspace found at %s but no projects were resolved", workspaceRoot)
	}

	selected, err := selectNxProject(projects, projectName, startDir)
	if err != nil {
		return Scope{}, err
	}

	return Scope{
		Name:          selected.Name,
		WorkspaceRoot: workspaceRoot,
		PrimaryRoot:   selected.Root,
		Roots:         []string{selected.Root},
		WorkspaceType: "nx",
	}, nil
}

func DiscoverNxProjectNames(path string) ([]string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	workspaceRoot := detectNxWorkspaceRoot(absPath)
	if workspaceRoot == "" {
		return nil, nil
	}

	projects, err := discoverNxProjects(workspaceRoot)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(projects))
	for _, project := range projects {
		names = append(names, project.Name)
	}
	sort.Strings(names)
	return names, nil
}

func detectNxWorkspaceRoot(startDir string) string {
	current := startDir
	for {
		if _, err := os.Stat(filepath.Join(current, "nx.json")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func discoverNxProjects(workspaceRoot string) ([]NxProject, error) {
	projectMap := map[string]string{}
	err := filepath.Walk(workspaceRoot, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", ".opencode", ".vectos", "coverage", "playwright-report", "test-results", "dist", ".next", "build":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if info.Name() != "project.json" {
			return nil
		}

		project, err := readNxProjectFile(current, workspaceRoot)
		if err != nil {
			return nil
		}
		projectMap[project.Name] = project.Root
		return nil
	})
	if err != nil {
		return nil, err
	}

	projects := make([]NxProject, 0, len(projectMap))
	for name, root := range projectMap {
		projects = append(projects, NxProject{Name: name, Root: root})
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})
	return projects, nil
}

func readNxProjectFile(projectFile string, workspaceRoot string) (NxProject, error) {
	content, err := os.ReadFile(projectFile)
	if err != nil {
		return NxProject{}, err
	}

	var disk struct {
		Name string `json:"name"`
		Root string `json:"root"`
	}
	if err := json.Unmarshal(content, &disk); err != nil {
		return NxProject{}, err
	}

	root := strings.TrimSpace(disk.Root)
	if root == "" {
		root = filepath.Dir(projectFile)
	} else {
		root = filepath.Join(workspaceRoot, filepath.FromSlash(root))
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return NxProject{}, err
	}

	name := strings.TrimSpace(disk.Name)
	if name == "" {
		name = filepath.Base(root)
	}

	return NxProject{Name: name, Root: root}, nil
}

func selectNxProject(projects []NxProject, requestedName string, startDir string) (NxProject, error) {
	if requestedName != "" {
		for _, project := range projects {
			if project.Name == requestedName {
				return project, nil
			}
		}
		return NxProject{}, fmt.Errorf("nx project %q not found", requestedName)
	}

	for _, project := range projects {
		if sameOrUnder(startDir, project.Root) {
			return project, nil
		}
	}

	if len(projects) == 1 {
		return projects[0], nil
	}

	names := make([]string, 0, len(projects))
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return NxProject{}, fmt.Errorf("multiple Nx projects detected; select one explicitly: %s", strings.Join(names, ", "))
}

func sameOrUnder(path string, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func detectProjectRoot(startDir string) string {
	markers := []string{".git", "go.mod", "package.json", "pyproject.toml", "Cargo.toml"}
	current := startDir

	for {
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(current, marker)); err == nil {
				return current
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			return startDir
		}
		current = parent
	}
}
