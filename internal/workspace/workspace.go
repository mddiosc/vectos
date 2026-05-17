package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const nxGraphPrintFlag = "--print"

type Scope struct {
	Name          string   `json:"name"`
	WorkspaceRoot string   `json:"workspace_root,omitempty"`
	PrimaryRoot   string   `json:"primary_root"`
	Roots         []string `json:"roots"`
	Warnings      []string `json:"warnings,omitempty"`
	WorkspaceType string   `json:"workspace_type,omitempty"`
}

func (s Scope) IsWorkspace() bool {
	return strings.TrimSpace(s.WorkspaceType) != ""
}

type NxProject struct {
	Name string `json:"name"`
	Root string `json:"root"`
}

type nxGraphDependency struct {
	Target string `json:"target"`
}

type nxGraphProject struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data struct {
		Root string `json:"root"`
	} `json:"data"`
}

type nxGraphData struct {
	Dependencies map[string][]nxGraphDependency `json:"dependencies"`
	Projects     []nxGraphProject               `json:"projects"`
	Graph        struct {
		Dependencies map[string][]nxGraphDependency `json:"dependencies"`
		Nodes        map[string]struct {
			Type string `json:"type"`
			Data struct {
				Root string `json:"root"`
			} `json:"data"`
		} `json:"nodes"`
	} `json:"graph"`
}

var nxGraphReader = readNxGraph

var nxGraphCache = struct {
	sync.RWMutex
	entries map[string]nxGraphData
}{entries: map[string]nxGraphData{}}

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

	selected, err := selectNxProject(projects, projectName, startDir, workspaceRoot)
	if err != nil {
		return Scope{}, err
	}

	roots := []string{selected.Root}
	var warnings []string
	if resolvedRoots, err := resolveNxProjectRoots(workspaceRoot, selected, projects); err == nil && len(resolvedRoots) > 0 {
		roots = resolvedRoots
	} else if err != nil {
		warnings = append(warnings, fmt.Sprintf("Nx graph resolution failed for %s: %v — falling back to primary project root only", selected.Name, err))
	}

	return Scope{
		Name:          selected.Name,
		WorkspaceRoot: workspaceRoot,
		PrimaryRoot:   selected.Root,
		Roots:         roots,
		Warnings:      warnings,
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

func DetectNxWorkspaceRoot(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	startDir := absPath
	if !info.IsDir() {
		startDir = filepath.Dir(absPath)
	}

	return detectNxWorkspaceRoot(startDir), nil
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

func resolveNxProjectRoots(workspaceRoot string, selected NxProject, projects []NxProject) ([]string, error) {
	projectMap := make(map[string]NxProject, len(projects))
	for _, project := range projects {
		projectMap[project.Name] = project
	}

	graph, err := loadNxGraph(workspaceRoot)
	if err != nil {
		return nil, err
	}

	dependencies, err := nxGraphDependencies(graph)
	if err != nil {
		return nil, err
	}
	excludedProjects := nxExcludedProjects(graph, projectMap)

	seenProjects := map[string]struct{}{selected.Name: {}}
	seenRoots := map[string]struct{}{selected.Root: {}}
	roots := []string{selected.Root}
	queue := []string{selected.Name}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		targets := internalDependencyTargets(dependencies[current], projectMap)
		for _, target := range targets {
			if _, excluded := excludedProjects[target]; excluded {
				continue
			}
			if _, seen := seenProjects[target]; seen {
				continue
			}
			seenProjects[target] = struct{}{}
			queue = append(queue, target)

			project := projectMap[target]
			if _, seen := seenRoots[project.Root]; seen {
				continue
			}
			seenRoots[project.Root] = struct{}{}
			roots = append(roots, project.Root)
		}
	}

	return roots, nil
}

func internalDependencyTargets(dependencies []nxGraphDependency, projectMap map[string]NxProject) []string {
	targets := make([]string, 0, len(dependencies))
	seen := make(map[string]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		if _, ok := projectMap[dependency.Target]; !ok {
			continue
		}
		if _, ok := seen[dependency.Target]; ok {
			continue
		}
		seen[dependency.Target] = struct{}{}
		targets = append(targets, dependency.Target)
	}
	sort.Strings(targets)
	return targets
}

func loadNxGraph(workspaceRoot string) (nxGraphData, error) {
	nxGraphCache.RLock()
	if graph, ok := nxGraphCache.entries[workspaceRoot]; ok {
		nxGraphCache.RUnlock()
		return graph, nil
	}
	nxGraphCache.RUnlock()

	graph, err := nxGraphReader(workspaceRoot)
	if err != nil {
		return nxGraphData{}, err
	}

	nxGraphCache.Lock()
	nxGraphCache.entries[workspaceRoot] = graph
	nxGraphCache.Unlock()

	return graph, nil
}

func nxGraphDependencies(graph nxGraphData) (map[string][]nxGraphDependency, error) {
	if len(graph.Dependencies) > 0 {
		return graph.Dependencies, nil
	}
	if len(graph.Graph.Dependencies) > 0 {
		return graph.Graph.Dependencies, nil
	}

	return nil, fmt.Errorf("nx graph output did not contain dependencies")
}

func nxExcludedProjects(graph nxGraphData, projectMap map[string]NxProject) map[string]struct{} {
	excluded := map[string]struct{}{}
	for _, project := range graph.Projects {
		if shouldExcludeNxProject(project.Name, project.Type, project.Data.Root, projectMap) {
			excluded[project.Name] = struct{}{}
		}
	}
	for name, node := range graph.Graph.Nodes {
		if shouldExcludeNxProject(name, node.Type, node.Data.Root, projectMap) {
			excluded[name] = struct{}{}
		}
	}
	return excluded
}

func shouldExcludeNxProject(name string, projectType string, graphRoot string, projectMap map[string]NxProject) bool {
	if os.Getenv("VECTOS_NX_INCLUDE_E2E") == "1" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(projectType), "e2e")
}

func readNxGraph(workspaceRoot string) (nxGraphData, error) {
	commands := nxGraphCommands(workspaceRoot)
	var lastErr error
	for _, args := range commands {
		output, err := runNxGraphCommand(workspaceRoot, args)
		if err != nil {
			lastErr = err
			continue
		}

		var graph nxGraphData
		if err := json.Unmarshal(output, &graph); err != nil {
			lastErr = err
			continue
		}
		return graph, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("nx graph command not available")
	}
	return nxGraphData{}, lastErr
}

func nxGraphCommands(workspaceRoot string) [][]string {
	commands := make([][]string, 0, 6)
	localNx := filepath.Join(workspaceRoot, "node_modules", ".bin", "nx")
	if _, err := os.Stat(localNx); err == nil {
		commands = append(commands, []string{localNx, "graph", nxGraphPrintFlag})
	}
	if _, err := os.Stat(localNx + ".cmd"); err == nil {
		commands = append(commands, []string{localNx + ".cmd", "graph", nxGraphPrintFlag})
	}
	commands = append(commands,
		[]string{"nx", "graph", nxGraphPrintFlag},
		[]string{"npx", "nx", "graph", nxGraphPrintFlag},
		[]string{"pnpm", "nx", "graph", nxGraphPrintFlag},
		[]string{"yarn", "nx", "graph", nxGraphPrintFlag},
		[]string{"bunx", "nx", "graph", nxGraphPrintFlag},
	)
	return commands
}

func runNxGraphCommand(workspaceRoot string, args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing nx graph command")
	}
	if _, err := exec.LookPath(args[0]); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = workspaceRoot
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

var ignoredNxDirs = map[string]struct{}{
	".git": {}, "node_modules": {}, ".opencode": {}, ".vectos": {},
	"coverage": {}, "playwright-report": {}, "test-results": {},
	"dist": {}, ".next": {}, "build": {},
}

func discoverNxProjects(workspaceRoot string) ([]NxProject, error) {
	projectMap := map[string]string{}
	err := walkNxProjectFiles(workspaceRoot, func(projectFile string) {
		addNxProjectFromFile(projectMap, projectFile, workspaceRoot)
	})
	if err != nil {
		return nil, err
	}
	return nxProjectsFromMap(projectMap), nil
}

func walkNxProjectFiles(workspaceRoot string, visit func(string)) error {
	return filepath.Walk(workspaceRoot, func(current string, info os.FileInfo, walkErr error) error {
		return handleNxWalkEntry(current, info, walkErr, visit)
	})
}

func handleNxWalkEntry(current string, info os.FileInfo, walkErr error, visit func(string)) error {
	if walkErr != nil {
		return walkErr
	}
	if info.IsDir() {
		if _, skip := ignoredNxDirs[info.Name()]; skip {
			return filepath.SkipDir
		}
		return nil
	}
	if info.Name() == "project.json" {
		visit(current)
	}
	return nil
}

func addNxProjectFromFile(projectMap map[string]string, projectFile string, workspaceRoot string) {
	project, err := readNxProjectFile(projectFile, workspaceRoot)
	if err != nil {
		return // malformed or unreadable project.json is silently skipped
	}
	projectMap[project.Name] = project.Root
}

func nxProjectsFromMap(projectMap map[string]string) []NxProject {
	projects := make([]NxProject, 0, len(projectMap))
	for name, root := range projectMap {
		projects = append(projects, NxProject{Name: name, Root: root})
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})
	return projects
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

func selectNxProject(projects []NxProject, requestedName string, startDir string, workspaceRoot string) (NxProject, error) {
	if requestedName != "" {
		for _, project := range projects {
			if project.Name == requestedName {
				return project, nil
			}
		}
		return NxProject{}, fmt.Errorf("nx project %q not found", requestedName)
	}

	// Check workspace-root ambiguity before containment matching so that a
	// project with root "." never silently wins when multiple projects exist.
	if startDir == workspaceRoot && len(projects) > 1 {
		return NxProject{}, fmt.Errorf("path is the Nx workspace root; please specify a project name. Available projects: %s", strings.Join(nxProjectNames(projects), ", "))
	}

	for _, project := range projects {
		if sameOrUnder(startDir, project.Root) {
			return project, nil
		}
	}

	if len(projects) == 1 {
		return projects[0], nil
	}

	return NxProject{}, fmt.Errorf("multiple Nx projects detected; select one explicitly: %s", strings.Join(nxProjectNames(projects), ", "))
}

func nxProjectNames(projects []NxProject) []string {
	names := make([]string, 0, len(projects))
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return names
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
