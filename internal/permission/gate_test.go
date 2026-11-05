package permission

import (
	"bufio"
	"strings"
	"testing"
)

func TestGateDeniesBlanketAlwaysForBashAndWriteFile(t *testing.T) {
	// Simulate user typing 'always' or 'a' for bash
	input := "always\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	gate := NewGate(scanner)

	// Bash: typing 'always' should approve ONCE, but NOT add to gate.always
	approved := gate.Ask("bash", `{"command": "dir"}`)
	if !approved {
		t.Errorf("expected bash to be approved for one execution")
	}

	if gate.always["bash"] {
		t.Errorf("gate.always['bash'] must remain false to prevent blanket approvals")
	}

	// WriteFile: typing 'a' should approve ONCE, but NOT add to gate.always
	scanner2 := bufio.NewScanner(strings.NewReader("a\n"))
	gate2 := NewGate(scanner2)
	approved2 := gate2.Ask("write_file", `{"path": "a.txt", "content": "hi"}`)
	if !approved2 {
		t.Errorf("expected write_file to be approved for one execution")
	}
	if gate2.always["write_file"] {
		t.Errorf("gate.always['write_file'] must remain false to prevent blanket approvals")
	}
}

func TestGateRequiresConfirmationForReadOnlyTools(t *testing.T) {
	// read_file and devops_status must NOT auto-approve without asking
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
