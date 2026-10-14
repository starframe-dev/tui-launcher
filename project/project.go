package project

import (
	"os"
	"path/filepath"
	"strings"
)

type Task struct {
	Name        string
	Description string
	Command     string
}

type TreeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Children []*TreeNode
}

type Project struct {
	Path  string
	Name  string
	Tasks []Task
	Tree  *TreeNode
}

func New(path string) *Project {
	return &Project{
		Path: path,
		Name: filepath.Base(path),
	}
}

func ScanForProjects(searchPaths []string) ([]*Project, error) {
	var projects []*Project
	depth := 2

	for _, basePath := range searchPaths {
		if err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			// Check depth
			rel, _ := filepath.Rel(basePath, path)
			if rel != "." && strings.Count(rel, string(filepath.Separator)) >= depth {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if !info.IsDir() {
				return nil
			}

			// Check for Taskfile.yml or package.json in this directory
			taskfilePath := filepath.Join(path, "Taskfile.yml")
			pkgPath := filepath.Join(path, "package.json")
			_, hasTaskfile := os.Stat(taskfilePath)
			_, hasPkg := os.Stat(pkgPath)
			if hasTaskfile != nil && hasPkg != nil {
				return nil
			}

			projects = append(projects, New(path))
			return filepath.SkipDir
		}); err != nil {
			return nil, err
		}
	}

	return projects, nil
}

func BuildTree(path string, maxDepth int) *TreeNode {
	root := &TreeNode{
		Name:     filepath.Base(path),
		Path:     path,
		IsDir:    true,
		Children: []*TreeNode{},
	}

	buildTreeRecursive(root, path, 0, maxDepth)
	return root
}

func buildTreeRecursive(node *TreeNode, dirPath string, currentDepth, maxDepth int) {
	if currentDepth >= maxDepth {
		return
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		// Skip hidden files and common ignored directories
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == ".git" {
			continue
		}

		childPath := filepath.Join(dirPath, entry.Name())
		child := &TreeNode{
			Name:     entry.Name(),
			Path:     childPath,
			IsDir:    entry.IsDir(),
			Children: []*TreeNode{},
		}

		if entry.IsDir() {
			buildTreeRecursive(child, childPath, currentDepth+1, maxDepth)
		}

		node.Children = append(node.Children, child)
	}
}
