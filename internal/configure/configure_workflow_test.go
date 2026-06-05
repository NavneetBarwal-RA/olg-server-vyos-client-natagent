package configure

import (
	"context"
	"testing"
	"time"

	"github.com/routerarchitects/nats-agent-core/agentcore"
	"github.com/routerarchitects/olg-server-vyos-client-natagent/internal/renderer"
	"github.com/routerarchitects/olg-server-vyos-client-natagent/internal/testutil"
)

func TestConfigureWorkflowSuccessAppliesAndSavesState(t *testing.T) {
	fixture := newPhase3WorkflowFixture(t, "cfg-phase3-success")

	if err := fixture.service.Handle(context.Background(), fixture.msg); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if got := fixture.renderer.Calls(); got != 1 {
		t.Fatalf("renderer calls got=%d want=1", got)
	}
	if got := fixture.apply.Calls(); got != 1 {
		t.Fatalf("apply calls got=%d want=1", got)
	}
	if got := fixture.store.SaveCalls(); got != 1 {
		t.Fatalf("save calls got=%d want=1", got)
	}
	saved, ok := fixture.store.LastSavedState()
	if !ok {
		t.Fatal("expected saved state")
	}
	if saved.Target != fixture.msg.Target || saved.AppliedUUID != fixture.msg.UUID {
		t.Fatalf("saved state got=%+v want target=%q uuid=%q", saved, fixture.msg.Target, fixture.msg.UUID)
	}
	if !fixture.client.ContainsStatus("success", "applied") {
		t.Fatal("expected applied success status")
	}
	if !fixture.client.ContainsResult("success", "configure") {
		t.Fatal("expected configure success result")
	}
	assertNoFailureResult(t, fixture.client)
}

func TestConfigureWorkflowSavesStateAfterApply(t *testing.T) {
	fixture := newPhase3WorkflowFixture(t, "cfg-phase3-order")

	if err := fixture.service.Handle(context.Background(), fixture.msg); err != nil {
		t.Fatalf("handle: %v", err)
	}

	assertEventOrder(t, fixture.events, "render", "apply")
	assertEventOrder(t, fixture.events, "apply", "state_save")
}

func TestConfigureWorkflowPublishesSuccessAfterStateSave(t *testing.T) {
	fixture := newPhase3WorkflowFixture(t, "cfg-phase3-publish")

	if err := fixture.service.Handle(context.Background(), fixture.msg); err != nil {
		t.Fatalf("handle: %v", err)
	}

	assertEventOrder(t, fixture.events, "apply", "state_save")
	assertEventOrder(t, fixture.events, "state_save", "publish_success")
}

func TestConfigureWorkflowAlreadyInSyncSkipsApply(t *testing.T) {
	fixture := newPhase3WorkflowFixture(t, "cfg-phase3-sync")
	fixture.store.Current.AppliedUUID = fixture.msg.UUID

	if err := fixture.service.Handle(context.Background(), fixture.msg); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if got := fixture.renderer.Calls(); got != 0 {
		t.Fatalf("renderer calls got=%d want=0", got)
	}
	if got := fixture.apply.Calls(); got != 0 {
		t.Fatalf("apply calls got=%d want=0", got)
	}
	if got := fixture.store.SaveCalls(); got != 0 {
		t.Fatalf("save calls got=%d want=0", got)
	}
	if !fixture.client.ContainsStatus("success", "already_in_sync") {
		t.Fatal("expected already_in_sync success status")
	}
	result, ok := fixture.client.LastResult()
	if !ok {
		t.Fatal("expected success result")
	}
	if result.Result != "success" || result.CommandType != "configure" || result.Message != "desired config already applied" {
		t.Fatalf("unexpected already-in-sync result: %+v", result)
	}
	assertNoFailureResult(t, fixture.client)
}

func TestConfigureWorkflowRepeatedSameUUIDIsIdempotent(t *testing.T) {
	fixture := newPhase3WorkflowFixture(t, "cfg-phase3-repeat")

	if err := fixture.service.Handle(context.Background(), fixture.msg); err != nil {
		t.Fatalf("first handle: %v", err)
	}
	if got := fixture.renderer.Calls(); got != 1 {
		t.Fatalf("renderer calls after first handle got=%d want=1", got)
	}
	if got := fixture.apply.Calls(); got != 1 {
		t.Fatalf("apply calls after first handle got=%d want=1", got)
	}
	if got := fixture.store.SaveCalls(); got != 1 {
		t.Fatalf("save calls after first handle got=%d want=1", got)
	}

	if err := fixture.service.Handle(context.Background(), fixture.msg); err != nil {
		t.Fatalf("second handle: %v", err)
	}
	if got := fixture.renderer.Calls(); got != 1 {
		t.Fatalf("renderer calls after second handle got=%d want still 1", got)
	}
	if got := fixture.apply.Calls(); got != 1 {
		t.Fatalf("apply calls after second handle got=%d want still 1", got)
	}
	if got := fixture.store.SaveCalls(); got != 1 {
		t.Fatalf("save calls after second handle got=%d want still 1", got)
	}
	if !fixture.client.ContainsStatus("success", "already_in_sync") {
		t.Fatal("expected already_in_sync status on repeated UUID")
	}
	results := fixture.client.Results()
	if len(results) != 2 {
		t.Fatalf("result count got=%d want=2", len(results))
	}
	if results[0].Result != "success" || results[1].Result != "success" {
		t.Fatalf("unexpected results: %+v", results)
	}
	if results[1].Message != "desired config already applied" {
		t.Fatalf("second result message got=%q want already applied", results[1].Message)
	}
	assertNoFailureResult(t, fixture.client)
}

func TestConfigureWorkflowInvalidDesiredConfigFailsBeforeSideEffects(t *testing.T) {
	fixture := newPhase3WorkflowFixture(t, "cfg-phase3-invalid")
	invalid := testutil.DesiredConfig(fixture.msg.Target, fixture.msg.UUID, testutil.InvalidPayload())
	fixture.client.Desired = &invalid

	err := fixture.service.Handle(context.Background(), fixture.msg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if got := fixture.client.LoadCalls(); got != 1 {
		t.Fatalf("desired load calls got=%d want=1", got)
	}
	if got := fixture.store.LoadCalls(); got != 0 {
		t.Fatalf("state load calls got=%d want=0", got)
	}
	if got := fixture.renderer.Calls(); got != 0 {
		t.Fatalf("renderer calls got=%d want=0", got)
	}
	if got := fixture.apply.Calls(); got != 0 {
		t.Fatalf("apply calls got=%d want=0", got)
	}
	if got := fixture.store.SaveCalls(); got != 0 {
		t.Fatalf("save calls got=%d want=0", got)
	}
	result, ok := fixture.client.LastResult()
	if !ok {
		t.Fatal("expected failure result")
	}
	if result.Result != "failure" || result.ErrorCode != "desired_payload_invalid" {
		t.Fatalf("failure result got=%+v want result=failure error_code=desired_payload_invalid", result)
	}
	if result.Target != fixture.msg.Target || result.UUID != fixture.msg.UUID || result.RPCID != fixture.msg.RPCID {
		t.Fatalf("failure result lost correlation data: %+v", result)
	}
}

func TestConfigureWorkflowEmptyUUIDFailsBeforeSideEffects(t *testing.T) {
	fixture := newPhase3WorkflowFixture(t, "cfg-phase3-empty")
	fixture.msg.UUID = ""
	invalid := testutil.DesiredConfig(fixture.msg.Target, "", testutil.MinimalDesiredConfig().Record.Payload)
	fixture.client.Desired = &invalid

	err := fixture.service.Handle(context.Background(), fixture.msg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if got := fixture.client.LoadCalls(); got != 1 {
		t.Fatalf("desired load calls got=%d want=1", got)
	}
	if got := fixture.store.LoadCalls(); got != 0 {
		t.Fatalf("state load calls got=%d want=0", got)
	}
	if got := fixture.renderer.Calls(); got != 0 {
		t.Fatalf("renderer calls got=%d want=0", got)
	}
	if got := fixture.apply.Calls(); got != 0 {
		t.Fatalf("apply calls got=%d want=0", got)
	}
	if got := fixture.store.SaveCalls(); got != 0 {
		t.Fatalf("save calls got=%d want=0", got)
	}
	result, ok := fixture.client.LastResult()
	if !ok {
		t.Fatal("expected failure result")
	}
	if result.Result != "failure" || result.ErrorCode != "desired_uuid_invalid" {
		t.Fatalf("failure result got=%+v want result=failure error_code=desired_uuid_invalid", result)
	}
	if result.Target != fixture.msg.Target || result.UUID != "" || result.RPCID != fixture.msg.RPCID {
		t.Fatalf("failure result lost correlation data: %+v", result)
	}
}

func TestConfigureWorkflowSuccessPreservesCorrelationIdentifiers(t *testing.T) {
	fixture := newPhase3WorkflowFixture(t, "cfg-phase3-correlation")

	if err := fixture.service.Handle(context.Background(), fixture.msg); err != nil {
		t.Fatalf("handle: %v", err)
	}

	result, ok := fixture.client.LastResult()
	if !ok {
		t.Fatal("expected result")
	}
	if result.RPCID != fixture.msg.RPCID || result.Target != fixture.msg.Target || result.UUID != fixture.msg.UUID {
		t.Fatalf("result correlation got rpc_id=%q target=%q uuid=%q", result.RPCID, result.Target, result.UUID)
	}
	input, ok := fixture.renderer.LastInput()
	if !ok {
		t.Fatal("expected renderer input")
	}
	if input.Record.Target != fixture.msg.Target || input.Record.UUID != fixture.msg.UUID {
		t.Fatalf("renderer input got target=%q uuid=%q", input.Record.Target, input.Record.UUID)
	}
	applied, ok := fixture.apply.LastInput()
	if !ok {
		t.Fatal("expected apply input")
	}
	if applied.Target != fixture.msg.Target || applied.UUID != fixture.msg.UUID {
		t.Fatalf("apply input got target=%q uuid=%q", applied.Target, applied.UUID)
	}
}

type phase3WorkflowFixture struct {
	msg      agentcore.ConfigureNotification
	client   *testutil.FakeConfigureClient
	store    *testutil.FakeStateStore
	renderer *testutil.FakeRenderer
	apply    *testutil.FakeApplyEngine
	events   *testutil.EventRecorder
	service  *Service
}

func newPhase3WorkflowFixture(t *testing.T, uuid string) phase3WorkflowFixture {
	t.Helper()

	events := &testutil.EventRecorder{}
	msg := testutil.MinimalConfigureNotification()
	msg.UUID = uuid
	msg.RPCID = "rpc-" + uuid

	desired := testutil.DesiredConfig(msg.Target, msg.UUID, testutil.MinimalDesiredConfig().Record.Payload)
	client := &testutil.FakeConfigureClient{
		Desired: &desired,
		Events:  events,
	}
	client.StatusRecorder.Events = events
	client.ResultRecorder.Events = events

	store := &testutil.FakeStateStore{Events: events}
	rndr := &testutil.FakeRenderer{
		Output: renderer.Output{
			Target: msg.Target,
			UUID:   msg.UUID,
			Text:   "set system host-name phase3\n",
		},
		UseOutput: true,
		Events:    events,
	}
	apply := &testutil.FakeApplyEngine{Events: events}

	svc, err := NewService(Dependencies{
		Client:      client,
		StateStore:  store,
		Renderer:    rndr,
		ApplyEngine: apply,
		Now: func() time.Time {
			return time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	return phase3WorkflowFixture{
		msg:      msg,
		client:   client,
		store:    store,
		renderer: rndr,
		apply:    apply,
		events:   events,
		service:  svc,
	}
}

func assertNoFailureResult(t *testing.T, recorder *testutil.FakeConfigureClient) {
	t.Helper()

	for _, result := range recorder.Results() {
		if result.Result == "failure" {
			t.Fatalf("unexpected failure result: %+v", result)
		}
	}
}

func assertEventOrder(t *testing.T, recorder *testutil.EventRecorder, before string, after string) {
	t.Helper()

	beforeIndex := recorder.Index(before)
	if beforeIndex < 0 {
		t.Fatalf("missing event %q in %v", before, recorder.Events())
	}
	afterIndex := recorder.Index(after)
	if afterIndex < 0 {
		t.Fatalf("missing event %q in %v", after, recorder.Events())
	}
	if beforeIndex >= afterIndex {
		t.Fatalf("event order got %q at %d and %q at %d in %v", before, beforeIndex, after, afterIndex, recorder.Events())
	}
}
