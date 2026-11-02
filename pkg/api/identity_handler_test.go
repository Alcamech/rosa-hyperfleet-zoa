package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openshift-online/rosa-seaplane-zoa/pkg/config"
	"github.com/openshift-online/rosa-seaplane-zoa/pkg/store"
)

func newHandlerForIdentityTests(sessionStore store.SessionStore) *Handler {
	cfg := &config.Config{TargetCluster: "test-rc"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := &Handler{
		cfg:          cfg,
		sessionStore: sessionStore,
		logger:       logger,
	}
	return h
}

func requestWithAuthenticatedCaller(callerARN string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v0/trusted-actions/get_pods/run", nil)
	return attachAuthenticatedIdentityForTest(r, callerARN, testCallerAccountID)
}

func TestHandler_resolveIdentity_WhenLaptopInvokerWithSessionStore_ItShouldReturnHumanOperator(t *testing.T) {
	h := newHandlerForIdentityTests(&mockSessionStore{
		getByTaskIDFn: func(_ context.Context, _ string) (*store.Session, error) {
			t.Fatal("task-id bridge must not run for human session name")
			return nil, nil
		},
	})
	arn := "arn:aws:sts::123456:assumed-role/sre-invoker/slopezma"
	op, signer, sess := h.resolveIdentity(requestWithAuthenticatedCaller(arn))
	if op != "slopezma" || signer != arn || sess != "" {
		t.Fatalf("got operator=%q signer=%q session=%q", op, signer, sess)
	}
}

func TestHandler_resolveIdentity_WhenBoundaryBridgeHit_ItShouldReturnSessionOperatorAndID(t *testing.T) {
	h := newHandlerForIdentityTests(&mockSessionStore{
		getByTaskIDFn: func(_ context.Context, taskID string) (*store.Session, error) {
			if taskID != testECSTaskID {
				return nil, fmt.Errorf("unexpected task id %q", taskID)
			}
			return &store.Session{
				SessionID: "sess-bridge-1",
				Operator:  "slopezma",
				TaskID:    taskID,
			}, nil
		},
	})
	arn := testBoundaryTaskARN()
	op, signer, sess := h.resolveIdentity(requestWithAuthenticatedCaller(arn))
	if op != "slopezma" || signer != arn || sess != "sess-bridge-1" {
		t.Fatalf("got operator=%q signer=%q session=%q", op, signer, sess)
	}
}

func TestHandler_resolveIdentity_WhenBoundaryBridgeMiss_ItShouldFallbackToTaskID(t *testing.T) {
	h := newHandlerForIdentityTests(&mockSessionStore{
		getByTaskIDFn: func(_ context.Context, _ string) (*store.Session, error) {
			return nil, nil
		},
	})
	arn := testBoundaryTaskARN()
	op, signer, sess := h.resolveIdentity(requestWithAuthenticatedCaller(arn))
	if op != testECSTaskID || signer != arn || sess != "" {
		t.Fatalf("got operator=%q signer=%q session=%q", op, signer, sess)
	}
}

func TestHandler_resolveIdentity_WhenBridgeLookupFails_ItShouldFallbackToARNExtraction(t *testing.T) {
	h := newHandlerForIdentityTests(&mockSessionStore{
		getByTaskIDFn: func(_ context.Context, _ string) (*store.Session, error) {
			return nil, fmt.Errorf("dynamodb unavailable")
		},
	})
	arn := testBoundaryTaskARN()
	op, signer, sess := h.resolveIdentity(requestWithAuthenticatedCaller(arn))
	if op != testECSTaskID || signer != arn || sess != "" {
		t.Fatalf("expected extract fallback, got operator=%q session=%q", op, sess)
	}
}

func TestHandler_resolveIdentity_WhenNoSessionStore_ItShouldExtractFromARNOnly(t *testing.T) {
	h := newHandlerForIdentityTests(nil)
	arn := "arn:aws:sts::123456:assumed-role/sre-role/slopezma"
	op, signer, sess := h.resolveIdentity(requestWithAuthenticatedCaller(arn))
	if op != "slopezma" || signer != arn || sess != "" {
		t.Fatalf("got operator=%q signer=%q session=%q", op, signer, sess)
	}
}

func TestHandler_resolveIdentity_WhenNoSessionStoreAndTaskRoleARN_ItShouldUseTaskIDAsOperator(t *testing.T) {
	h := newHandlerForIdentityTests(nil)
	arn := testBoundaryTaskARN()
	op, signer, sess := h.resolveIdentity(requestWithAuthenticatedCaller(arn))
	if op != testECSTaskID || signer != arn || sess != "" {
		t.Fatalf("got operator=%q signer=%q session=%q", op, signer, sess)
	}
}

func TestHandler_resolveIdentity_WhenWrongBoundaryRoleName_ItShouldSkipBridgeDespiteHexSession(t *testing.T) {
	called := false
	h := newHandlerForIdentityTests(&mockSessionStore{
		getByTaskIDFn: func(_ context.Context, _ string) (*store.Session, error) {
			called = true
			return &store.Session{Operator: "other", SessionID: "s-1"}, nil
		},
	})
	arn := "arn:aws:sts::123456:assumed-role/sre-invoker/" + testECSTaskID
	op, signer, sess := h.resolveIdentity(requestWithAuthenticatedCaller(arn))
	if called {
		t.Fatal("bridge must not run for non-boundary role")
	}
	if op != testECSTaskID || signer != arn || sess != "" {
		t.Fatalf("got operator=%q signer=%q session=%q", op, signer, sess)
	}
}

func TestHandler_resolveIdentity_WhenMissingAuthenticatedCaller_ItShouldReturnEmptyIdentity(t *testing.T) {
	h := newHandlerForIdentityTests(&mockSessionStore{})
	op, signer, sess := h.resolveIdentity(requestWithAuthenticatedCaller(""))
	if op != "" || signer != "" || sess != "" {
		t.Fatalf("got operator=%q signer=%q session=%q", op, signer, sess)
	}
}

func TestHandler_resolveIdentity_WhenInvalidARN_ItShouldUseRawARNAsOperator(t *testing.T) {
	h := newHandlerForIdentityTests(nil)
	raw := "not-an-arn"
	op, signer, sess := h.resolveIdentity(requestWithAuthenticatedCaller(raw))
	if op != raw || signer != raw || sess != "" {
		t.Fatalf("got operator=%q signer=%q session=%q", op, signer, sess)
	}
}

type capturingAuditStore struct {
	last *store.AuditEntry
}

func (c *capturingAuditStore) Record(_ context.Context, e *store.AuditEntry) error {
	c.last = e
	return nil
}

func (c *capturingAuditStore) List(_ context.Context, _ string, _ *store.AuditFilter) ([]*store.AuditEntry, error) {
	return nil, nil
}

func (c *capturingAuditStore) ListAll(_ context.Context, _ *store.AuditFilter) ([]*store.AuditEntry, error) {
	return nil, nil
}

func TestHandler_recordAudit_WhenBoundaryBridgeHit_ItShouldPersistOperatorSignerAndSession(t *testing.T) {
	audit := &capturingAuditStore{}
	h := newHandlerForIdentityTests(&mockSessionStore{
		getByTaskIDFn: func(_ context.Context, _ string) (*store.Session, error) {
			return &store.Session{SessionID: "sess-audit", Operator: "slopezma"}, nil
		},
	})
	h.auditStore = audit

	taskARN := testBoundaryTaskARN()
	r := attachAuthenticatedIdentityForTest(httptest.NewRequest(http.MethodPost, "/", nil), taskARN, "111122223333")
	h.recordAudit(r, http.StatusOK, "get_pods", "exec-1")

	if audit.last == nil {
		t.Fatal("expected audit record")
	}
	if audit.last.Operator != "slopezma" {
		t.Errorf("operator=%q", audit.last.Operator)
	}
	if audit.last.SessionID != "sess-audit" {
		t.Errorf("session_id=%q", audit.last.SessionID)
	}
	if audit.last.SignerARN == "" || audit.last.AccountID != "111122223333" {
		t.Errorf("signer_arn=%q account_id=%q", audit.last.SignerARN, audit.last.AccountID)
	}
}
