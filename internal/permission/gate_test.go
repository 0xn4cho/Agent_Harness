package permission

import (
	"bufio"
	"strings"
	"testing"
)

func TestGateDeniesBlanketAlwaysForBashWriteFileAndDevOps(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("always\n"))
	gate := NewGate(scanner)
	if !gate.Ask("bash", `{"command": "dir"}`) {
		t.Errorf("expected allowlisted bash 'dir' to be approved for one execution")
	}
	if gate.always["bash"] {
		t.Errorf("gate.always['bash'] must remain false to prevent blanket approvals")
	}

	scanner2 := bufio.NewScanner(strings.NewReader("a\n"))
	gate2 := NewGate(scanner2)
	if !gate2.Ask("write_file", `{"path": "a.txt", "content": "hi"}`) {
		t.Errorf("expected write_file to be approved for one execution")
	}
	if gate2.always["write_file"] {
		t.Errorf("gate.always['write_file'] must remain false to prevent blanket approvals")
	}

	scanner3 := bufio.NewScanner(strings.NewReader("always\n"))
	gate3 := NewGate(scanner3)
	if !gate3.Ask("devops_status", `{"target": "k8s"}`) {
		t.Errorf("expected devops_status to be approved for one execution")
	}
	if gate3.always["devops_status"] {
		t.Errorf("gate.always['devops_status'] must remain false to enforce per-operation confirmation")
	}
}

func TestGateReinforcedApprovalForFullShellBash(t *testing.T) {
	scannerDenyY := bufio.NewScanner(strings.NewReader("y\n"))
	gateDenyY := NewGate(scannerDenyY)
	if gateDenyY.Ask("bash", `{"command": "Remove-Item foo.txt"}`) {
		t.Errorf("expected single-letter 'y' to be rejected for high-risk full-shell bash command")
	}

	scannerAllowYes := bufio.NewScanner(strings.NewReader("yes\n"))
	gateAllowYes := NewGate(scannerAllowYes)
	if !gateAllowYes.Ask("bash", `{"command": "Remove-Item foo.txt"}`) {
		t.Errorf("expected full 'yes' to approve high-risk full-shell bash command")
	}

	scannerBlocked := bufio.NewScanner(strings.NewReader("yes\n"))
	gateBlocked := NewGate(scannerBlocked)
	if gateBlocked.Ask("bash", `{"command": "cat ~/.ssh/id_rsa"}`) {
		t.Errorf("expected blocked command to be denied even if user typed 'yes'")
	}
}

func TestGateRequiresConfirmationForReadOnlyTools(t *testing.T) {
	input := "n\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	gate := NewGate(scanner)

	approved := gate.Ask("read_file", `{"path": "foo.txt"}`)
	if approved {
		t.Errorf("read_file should NOT be auto-approved when user says 'n'")
	}

	scanner2 := bufio.NewScanner(strings.NewReader("n\n"))
	gate2 := NewGate(scanner2)
	approved2 := gate2.Ask("devops_status", `{"target": "all"}`)
	if approved2 {
		t.Errorf("devops_status should NOT be auto-approved when user says 'n'")
	}
}
