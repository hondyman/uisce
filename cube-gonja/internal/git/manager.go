package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cube-gonja/config"
)

type Manager struct {
	config  config.Config
	repoDir string
}

type CommitInfo struct {
	Hash      string    `json:"hash"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
}

func NewManager(cfg config.Config, repoDir string) *Manager {
	return &Manager{
		config:  cfg,
		repoDir: repoDir,
	}
}

func (m *Manager) InitializeRepo() error {
	if !m.config.GitEnabled {
		return nil
	}

	// Check if already a git repo
	if _, err := os.Stat(filepath.Join(m.repoDir, ".git")); err == nil {
		return nil // Already initialized
	}

	// Initialize git repo
	if err := m.runGitCommand("init"); err != nil {
		return fmt.Errorf("git init: %v", err)
	}

	// Set up initial config
	if err := m.runGitCommand("config", "user.name", "Gonja Service"); err != nil {
		return fmt.Errorf("git config user.name: %v", err)
	}

	if err := m.runGitCommand("config", "user.email", "gonja@service.local"); err != nil {
		return fmt.Errorf("git config user.email: %v", err)
	}

	// Add remote if specified
	if m.config.GitRemoteURL != "" {
		if err := m.runGitCommand("remote", "add", "origin", m.config.GitRemoteURL); err != nil {
			return fmt.Errorf("git remote add: %v", err)
		}
	}

	// Initial commit
	if err := m.AddAll(); err != nil {
		return fmt.Errorf("initial add: %v", err)
	}

	if err := m.Commit("Initial commit - Gonja templates and configuration"); err != nil {
		return fmt.Errorf("initial commit: %v", err)
	}

	return nil
}

func (m *Manager) AddAll() error {
	return m.runGitCommand("add", ".")
}

func (m *Manager) Commit(message string) error {
	return m.runGitCommand("commit", "-m", message)
}

func (m *Manager) Push() error {
	if m.config.GitRemoteURL == "" {
		return fmt.Errorf("no remote URL configured")
	}
	return m.runGitCommand("push", "-u", "origin", m.config.GitBranch)
}

func (m *Manager) Pull() error {
	if m.config.GitRemoteURL == "" {
		return fmt.Errorf("no remote URL configured")
	}
	return m.runGitCommand("pull", "origin", m.config.GitBranch)
}

func (m *Manager) GetStatus() (string, error) {
	return m.runGitCommandOutput("status", "--porcelain")
}

func (m *Manager) GetLog(limit int) ([]CommitInfo, error) {
	output, err := m.runGitCommandOutput("log", fmt.Sprintf("--max-count=%d", limit),
		"--pretty=format:%H|%s|%an|%ad", "--date=iso")
	if err != nil {
		return nil, err
	}

	var commits []CommitInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			timestamp, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[3])
			commits = append(commits, CommitInfo{
				Hash:      parts[0],
				Message:   parts[1],
				Author:    parts[2],
				Timestamp: timestamp,
			})
		}
	}

	return commits, nil
}

func (m *Manager) CreateBranch(branchName string) error {
	return m.runGitCommand("checkout", "-b", branchName)
}

func (m *Manager) SwitchBranch(branchName string) error {
	return m.runGitCommand("checkout", branchName)
}

func (m *Manager) GetBranches() ([]string, error) {
	output, err := m.runGitCommandOutput("branch", "-a")
	if err != nil {
		return nil, err
	}

	var branches []string
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "* ")
		if line != "" {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

func (m *Manager) runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = m.repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) runGitCommandOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = m.repoDir
	output, err := cmd.Output()
	return string(output), err
}
