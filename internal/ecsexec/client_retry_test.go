package ecsexec

import (
	"errors"
	"fmt"
	"testing"

	smithy "github.com/aws/smithy-go"
)

type stubAPIError struct {
	code    string
	message string
}

func (e stubAPIError) Error() string   { return e.message }
func (e stubAPIError) ErrorCode() string { return e.code }
func (e stubAPIError) ErrorMessage() string { return e.message }
func (e stubAPIError) ErrorFault() smithy.ErrorFault { return smithy.FaultUnknown }

func TestIsExecuteCommandRetryable_WhenTargetNotConnected_ItShouldReturnTrue(t *testing.T) {
	err := fmt.Errorf("ExecuteCommand: %w", stubAPIError{code: "TargetNotConnectedException", message: "not connected"})
	if !isExecuteCommandRetryable(err) {
		t.Fatal("expected retryable")
	}
}

func TestIsExecuteCommandRetryable_WhenAccessDenied_ItShouldReturnFalse(t *testing.T) {
	err := fmt.Errorf("ExecuteCommand: %w", stubAPIError{code: "AccessDeniedException", message: "denied"})
	if isExecuteCommandRetryable(err) {
		t.Fatal("expected not retryable")
	}
}

func TestIsExecuteCommandRetryable_WhenWrappedMessageContainsTargetNotConnected_ItShouldReturnTrue(t *testing.T) {
	err := errors.New("ExecuteCommand: operation error ECS: ExecuteCommand, TargetNotConnectedException: The execute command failed")
	if !isExecuteCommandRetryable(err) {
		t.Fatal("expected retryable from message")
	}
}
