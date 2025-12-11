package session

import (
	"os/exec"
	"strings"
	"testing"
)

// TestForkFlow_Integration tests the complete fork flow
// This is a longer-running integration test that requires tmux
func TestForkFlow_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create parent session with claude tool (gets pre-assigned ID)
	parent := NewInstanceWithTool("fork-parent", "/tmp", "claude")

	// Verify parent has pre-assigned session ID
	if parent.ClaudeSessionID == "" {
		t.Fatal("Parent should have pre-assigned ClaudeSessionID")
	}
	parentID := parent.ClaudeSessionID
	t.Logf("Parent session ID: %s", parentID)

	// Verify CanFork is true
	if !parent.CanFork() {
		t.Fatal("Parent should be able to fork")
	}

	// Create forked instance
	forked, cmd, err := parent.CreateForkedInstance("fork-child", "")
	if err != nil {
		t.Fatalf("CreateForkedInstance failed: %v", err)
	}

	// Verify fork command structure
	if strings.Contains(cmd, "--session-id") {
		t.Errorf("Fork command should NOT have --session-id: %s", cmd)
	}
	if !strings.Contains(cmd, "--resume "+parentID) {
		t.Errorf("Fork command should have --resume %s: %s", parentID, cmd)
	}
	if !strings.Contains(cmd, "--fork-session") {
		t.Errorf("Fork command should have --fork-session: %s", cmd)
	}

	// Verify forked instance state
	if forked.ClaudeSessionID != "" {
		t.Errorf("Forked instance should have empty session ID initially: %s", forked.ClaudeSessionID)
	}
	if forked.Tool != "claude" {
		t.Errorf("Forked tool = %s, want claude", forked.Tool)
	}
	if forked.ProjectPath != "/tmp" {
		t.Errorf("Forked path = %s, want /tmp", forked.ProjectPath)
	}

	t.Log("Fork flow test passed - fork command is correctly structured")
}

// TestMultipleSessionsSameProject tests that multiple sessions in same project
// get different session IDs
func TestMultipleSessionsSameProject(t *testing.T) {
	// Create two sessions in the same project directory
	session1 := NewInstanceWithTool("test1", "/tmp/same-project", "claude")
	session2 := NewInstanceWithTool("test2", "/tmp/same-project", "claude")

	// Both should have session IDs
	if session1.ClaudeSessionID == "" {
		t.Error("session1 should have ClaudeSessionID")
	}
	if session2.ClaudeSessionID == "" {
		t.Error("session2 should have ClaudeSessionID")
	}

	// Session IDs should be DIFFERENT
	if session1.ClaudeSessionID == session2.ClaudeSessionID {
		t.Errorf("Sessions in same project should have DIFFERENT IDs: %s == %s",
			session1.ClaudeSessionID, session2.ClaudeSessionID)
	}

	t.Logf("Session 1 ID: %s", session1.ClaudeSessionID)
	t.Logf("Session 2 ID: %s", session2.ClaudeSessionID)
}
