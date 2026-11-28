package mcp_server //nolint:revive // fine for now

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tasks (io.modelcontextprotocol/tasks, ext-tasks v0.2.3): task-augmented
// tools/call for native IaC runs, with tasks/get, tasks/update and
// tasks/cancel over an in-memory store.  go-sdk v1.8.0 has no tasks support
// and a tool handler cannot return resultType "task", so tasksMiddleware puts
// a taskSlot in the context of each eligible tools/call; the handler fills it
// when it goes async and the middleware returns the CreateTaskResult in place
// of the handler's result.  tasksMiddleware is the only code that writes
// resultType "task".  Not implemented: notifications/tasks, tasks/list and
// tasks/result, which Claude Code does not use.

type taskStatus string

const (
	taskStatusWorking   taskStatus = "working"
	taskStatusCompleted taskStatus = "completed"
	taskStatusFailed    taskStatus = "failed"
)

const (
	// taskIDBytes of crypto/rand, base64url without padding: 192 bits in 32
	// printable characters.
	taskIDBytes = 24
	// stdioPrincipal owns every task on stdio, where the process is the caller.
	stdioPrincipal = "stdio"
	// taskTimeoutSetting names the limit the task path applies.
	taskTimeoutSetting = "server.extensions.tasks.timeout"
)

// taskError is the JSON-RPC error object a failed task carries.
type taskError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// taskSnapshot is the data carrier for one task's state at one moment.
type taskSnapshot interface {
	ID() string
	Status() taskStatus
	StatusMessage() string
	CreatedAt() time.Time
	LastUpdatedAt() time.Time
	// Result is the tool result of a completed task.
	Result() *mcp.CallToolResult
	// Failure is the error of a failed task.
	Failure() *taskError
}

type standardTaskSnapshot struct {
	id            string
	status        taskStatus
	statusMessage string
	createdAt     time.Time
	lastUpdatedAt time.Time
	result        *mcp.CallToolResult
	failure       *taskError
}

func newTaskSnapshot(e *taskEntry) taskSnapshot {
	return &standardTaskSnapshot{
		id:            e.id,
		status:        e.status,
		statusMessage: e.statusMessage,
		createdAt:     e.createdAt,
		lastUpdatedAt: e.updatedAt,
		result:        e.result,
		failure:       e.failure,
	}
}

func (s *standardTaskSnapshot) ID() string                  { return s.id }
func (s *standardTaskSnapshot) Status() taskStatus          { return s.status }
func (s *standardTaskSnapshot) StatusMessage() string       { return s.statusMessage }
func (s *standardTaskSnapshot) CreatedAt() time.Time        { return s.createdAt }
func (s *standardTaskSnapshot) LastUpdatedAt() time.Time    { return s.lastUpdatedAt }
func (s *standardTaskSnapshot) Result() *mcp.CallToolResult { return s.result }
func (s *standardTaskSnapshot) Failure() *taskError         { return s.failure }

// taskStore keeps tasks for tasks/get, tasks/update and tasks/cancel.  The
// interface stays narrow so that a durable store keyed by IaC run_id can
// replace the in-memory one.
type taskStore interface {
	// create registers a working task; it is readable before create returns.
	create(owner, statusMessage string) (taskSnapshot, error)
	// get returns a task that exists, has not expired and belongs to owner.
	get(id, owner string) (taskSnapshot, bool)
	// finish moves a working task to completed (failure nil) or failed.
	finish(id string, result *mcp.CallToolResult, failure *taskError)
	// requestCancel records a cancellation request on a task owned by owner.
	requestCancel(id, owner string) bool
	// discard removes a task whose work never started.
	discard(id string)
}

type taskEntry struct {
	id              string
	owner           string
	status          taskStatus
	statusMessage   string
	cancelRequested bool
	createdAt       time.Time
	updatedAt       time.Time
	result          *mcp.CallToolResult
	failure         *taskError
}

// memoryTaskStore holds tasks in process memory.  A working task expires
// lifetime (tasks.timeout plus tasks.retention, the advertised ttlMs) after
// creation; a terminal task expires retention after it finished.  Expired
// tasks are swept on every access.  Tasks do not survive a restart and are
// not shared between HTTP instances.
type memoryTaskStore struct {
	mu        sync.Mutex
	tasks     map[string]*taskEntry
	lifetime  time.Duration
	retention time.Duration
	now       func() time.Time
}

func newMemoryTaskStore(lifetime, retention time.Duration, now func() time.Time) *memoryTaskStore {
	return &memoryTaskStore{tasks: map[string]*taskEntry{}, lifetime: lifetime, retention: retention, now: now}
}

func newTaskID() (string, error) {
	b := make([]byte, taskIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("task id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *memoryTaskStore) sweepLocked(now time.Time) {
	for id, e := range s.tasks {
		if e.status == taskStatusWorking {
			if now.Sub(e.createdAt) > s.lifetime {
				delete(s.tasks, id)
			}
		} else if now.Sub(e.updatedAt) > s.retention {
			delete(s.tasks, id)
		}
	}
}

func (s *memoryTaskStore) create(owner, statusMessage string) (taskSnapshot, error) {
	id, err := newTaskID()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	s.sweepLocked(now)
	e := &taskEntry{id: id, owner: owner, status: taskStatusWorking, statusMessage: statusMessage,
		createdAt: now, updatedAt: now}
	s.tasks[id] = e
	return newTaskSnapshot(e), nil
}

func (s *memoryTaskStore) get(id, owner string) (taskSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked(s.now())
	e, ok := s.tasks[id]
	if !ok || e.owner != owner {
		return nil, false
	}
	return newTaskSnapshot(e), true
}

func (s *memoryTaskStore) finish(id string, result *mcp.CallToolResult, failure *taskError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.tasks[id]
	if !ok || e.status != taskStatusWorking {
		return
	}
	e.updatedAt = s.now()
	note := ""
	if e.cancelRequested {
		note = "cancellation was requested but the run was not interrupted; "
	}
	switch {
	case failure != nil:
		e.status, e.failure, e.statusMessage = taskStatusFailed, failure, note+failure.Message
	case result != nil && result.IsError:
		e.status, e.result, e.statusMessage = taskStatusCompleted, result, note+"completed with a tool error"
	default:
		e.status, e.result, e.statusMessage = taskStatusCompleted, result, note+"completed"
	}
}

func (s *memoryTaskStore) requestCancel(id, owner string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	s.sweepLocked(now)
	e, ok := s.tasks[id]
	if !ok || e.owner != owner {
		return false
	}
	if e.status == taskStatusWorking && !e.cancelRequested {
		e.cancelRequested = true
		e.updatedAt = now
		e.statusMessage = "cancellation requested; the run is not interrupted, because stopping mid-run " +
			"can leave a partial deployment, and it reaches its own terminal status"
	}
	return true
}

func (s *memoryTaskStore) discard(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, id)
}

// Wire shapes.

type taskWire struct {
	TaskID         string `json:"taskId"`
	Status         string `json:"status"`
	StatusMessage  string `json:"statusMessage,omitempty"`
	CreatedAt      string `json:"createdAt"`
	LastUpdatedAt  string `json:"lastUpdatedAt"`
	TTLMs          int64  `json:"ttlMs"`
	PollIntervalMs int64  `json:"pollIntervalMs"`
}

// createTaskResult answers a tools/call that became a task.
type createTaskResult struct {
	mcp.ResultBase
	ResultType string `json:"resultType"`
	taskWire
}

type getTaskResult struct {
	mcp.ResultBase
	ResultType string `json:"resultType"`
	taskWire
	Result *mcp.CallToolResult `json:"result,omitempty"`
	Error  *taskError          `json:"error,omitempty"`
}

type taskAckResult struct {
	mcp.ResultBase
	ResultType string `json:"resultType"`
}

type taskIDParams struct {
	mcp.ParamsBase
	TaskID string `json:"taskId"`
}

type taskUpdateParams struct {
	mcp.ParamsBase
	TaskID         string         `json:"taskId"`
	InputResponses map[string]any `json:"inputResponses,omitempty"`
}

// taskManager runs and serves tasks for one server.
type taskManager struct {
	store        taskStore
	backend      Backend
	ttl          time.Duration
	pollInterval time.Duration
}

// newTaskManager builds the task path.  backend is the limited backend of the
// synchronous path: the task path shares its concurrency slots but applies
// tasks.timeout in place of server.request_timeout.
func newTaskManager(cfg TasksExtensionConfig, backend Backend) *taskManager {
	return &taskManager{
		store:        newMemoryTaskStore(cfg.GetTimeout()+cfg.GetRetention(), cfg.GetRetention(), time.Now),
		backend:      withTaskTimeout(backend, cfg.GetTimeout()),
		ttl:          cfg.GetTimeout() + cfg.GetRetention(),
		pollInterval: cfg.GetPollInterval(),
	}
}

func (m *taskManager) wire(s taskSnapshot) taskWire {
	return taskWire{
		TaskID:         s.ID(),
		Status:         string(s.Status()),
		StatusMessage:  s.StatusMessage(),
		CreatedAt:      s.CreatedAt().UTC().Format(time.RFC3339),
		LastUpdatedAt:  s.LastUpdatedAt().UTC().Format(time.RFC3339),
		TTLMs:          m.ttl.Milliseconds(),
		PollIntervalMs: m.pollInterval.Milliseconds(),
	}
}

func unknownTask(id string) error {
	return invalidParams("unknown task %q: it does not exist, has expired or belongs to another caller", id)
}

// register adds tasks/get, tasks/update, tasks/cancel and the middleware.
func (m *taskManager) register(server *mcp.Server) error {
	if err := mcp.AddReceivingCustomMethod(server, "tasks/get",
		func(ctx context.Context, _ *mcp.ServerSession, p *taskIDParams) (*getTaskResult, error) {
			id := taskIDOf(p)
			s, ok := m.store.get(id, taskPrincipalFrom(ctx))
			if !ok {
				return nil, unknownTask(id)
			}
			return &getTaskResult{ResultType: "complete", taskWire: m.wire(s), Result: s.Result(), Error: s.Failure()}, nil
		}); err != nil {
		return err
	}
	if err := mcp.AddReceivingCustomMethod(server, "tasks/update",
		func(ctx context.Context, _ *mcp.ServerSession, p *taskUpdateParams) (*taskAckResult, error) {
			var id string
			if p != nil {
				id = p.TaskID
			}
			// No task here enters input_required, so every response key is
			// one the task is not waiting for and is ignored.
			if _, ok := m.store.get(id, taskPrincipalFrom(ctx)); !ok {
				return nil, unknownTask(id)
			}
			return &taskAckResult{ResultType: "complete"}, nil
		}); err != nil {
		return err
	}
	if err := mcp.AddReceivingCustomMethod(server, "tasks/cancel",
		func(ctx context.Context, _ *mcp.ServerSession, p *taskIDParams) (*taskAckResult, error) {
			id := taskIDOf(p)
			if !m.store.requestCancel(id, taskPrincipalFrom(ctx)) {
				return nil, unknownTask(id)
			}
			return &taskAckResult{ResultType: "complete"}, nil
		}); err != nil {
		return err
	}
	server.AddReceivingMiddleware(m.middleware)
	return nil
}

func taskIDOf(p *taskIDParams) string {
	if p == nil {
		return ""
	}
	return p.TaskID
}

// middleware records the caller on every request and, on a tools/call whose
// client declares the tasks extension, offers the handler a taskSlot.  A call
// from a client that did not declare it never gets a slot, so it never
// receives a task.  Added after mcp.NewServer, it wraps the SDK's approval
// round trip, so approval completes before the handler can go async.
func (m *taskManager) middleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		ctx = context.WithValue(ctx, taskPrincipalKey{}, requestPrincipal(ctx, req.GetExtra()))
		if method != "tools/call" || !clientDeclaresExtension(clientCapabilitiesOf(req), tasksExtensionID) {
			return next(ctx, method, req)
		}
		slot := &taskSlot{manager: m, owner: taskPrincipalFrom(ctx)}
		res, err := next(context.WithValue(ctx, taskSlotKey{}, slot), method, req)
		if slot.created != nil {
			// The task is running; whatever the handler returned is a placeholder.
			return slot.created, nil
		}
		return res, err
	}
}

func clientCapabilitiesOf(req mcp.Request) *mcp.ClientCapabilities {
	if r, ok := req.(interface {
		ClientCapabilities() *mcp.ClientCapabilities
	}); ok {
		return r.ClientCapabilities()
	}
	return nil
}

type taskSlotKey struct{}

type taskPrincipalKey struct{}

// taskSlotFrom returns the slot of a task-eligible tools/call, or nil.
func taskSlotFrom(ctx context.Context) *taskSlot {
	slot, _ := ctx.Value(taskSlotKey{}).(*taskSlot)
	return slot
}

func taskPrincipalFrom(ctx context.Context) string {
	p, _ := ctx.Value(taskPrincipalKey{}).(string)
	return p
}

// taskAuditFunc writes one audit record for a task: status working when the
// task starts, its terminal status when the work finishes.
type taskAuditFunc func(ctx context.Context, taskID string, status taskStatus, toolErr error) error

// taskSlot hands one tools/call from the tool handler to the middleware.  The
// handler prepares it (the task exists and is readable from then on); the
// gate audits the decision and launches it; the middleware returns created.
type taskSlot struct {
	manager *taskManager
	owner   string
	task    taskSnapshot
	work    func(context.Context, Backend) (*mcp.CallToolResult, error)
	created *createTaskResult
}

// prepare registers the task.  work runs later against the task backend and
// returns the tool result, or the error the synchronous path would return.
func (s *taskSlot) prepare(work func(context.Context, Backend) (*mcp.CallToolResult, error)) error {
	task, err := s.manager.store.create(s.owner, "running in the background; poll tasks/get for the result")
	if err != nil {
		return err
	}
	s.task, s.work = task, work
	return nil
}

func (s *taskSlot) prepared() bool { return s != nil && s.task != nil && s.created == nil }

// launch writes the decision record, then starts the work on a context
// detached from the request.  If the decision cannot be audited the task is
// discarded and never runs.
func (s *taskSlot) launch(ctx context.Context, record taskAuditFunc) error {
	id := s.task.ID()
	if err := record(ctx, id, taskStatusWorking, nil); err != nil {
		s.manager.store.discard(id)
		return err
	}
	s.created = &createTaskResult{ResultType: "task", taskWire: s.manager.wire(s.task)}
	runCtx := context.WithoutCancel(ctx)
	go s.run(runCtx, record)
	return nil
}

func (s *taskSlot) run(ctx context.Context, record taskAuditFunc) {
	result, failure, toolErr := runTaskWork(ctx, s.manager.backend, s.work)
	status := taskStatusCompleted
	if failure != nil {
		status = taskStatusFailed
	}
	if err := record(ctx, s.task.ID(), status, toolErr); err != nil {
		// A strict failure_mode turns an unaudited completion into a failure,
		// as on the synchronous path.
		result, failure = nil, &taskError{Code: jsonrpc.CodeInternalError, Message: err.Error()}
	}
	s.manager.store.finish(s.task.ID(), result, failure)
}

// runTaskWork mirrors the SDK's handling of a typed tool handler: an ordinary
// error becomes an isError tool result (so the task completes with the result
// a synchronous call would have returned), a JSON-RPC error fails the task,
// and a panic fails it with an internal error.
func runTaskWork(
	ctx context.Context, backend Backend, work func(context.Context, Backend) (*mcp.CallToolResult, error),
) (*mcp.CallToolResult, *taskError, error) {
	var res *mcp.CallToolResult
	var err error
	var panicked any
	func() {
		defer func() { panicked = recover() }()
		res, err = work(ctx, backend)
	}()
	if panicked != nil {
		err = fmt.Errorf("task panicked: %v", panicked)
		return nil, &taskError{Code: jsonrpc.CodeInternalError, Message: err.Error()}, err
	}
	if err == nil {
		return res, nil, nil
	}
	var wireErr *jsonrpc.Error
	if errors.As(err, &wireErr) {
		return nil, &taskError{Code: wireErr.Code, Message: wireErr.Message, Data: wireErr.Data}, err
	}
	var errRes mcp.CallToolResult
	errRes.SetError(err)
	return &errRes, nil, err
}

// requestPrincipal names the caller that owns per-caller state.  OAuth uses
// the token subject; the other HTTP mechanisms use what secureHTTPHandler
// recorded; stdio has one caller.
func requestPrincipal(ctx context.Context, extra *mcp.RequestExtra) string {
	if extra != nil && extra.TokenInfo != nil && extra.TokenInfo.UserID != "" {
		return "oauth:" + extra.TokenInfo.UserID
	}
	if p, ok := ctx.Value(httpPrincipalKey{}).(string); ok && p != "" {
		return p
	}
	if extra == nil {
		return stdioPrincipal
	}
	return "http"
}

type httpPrincipalKey struct{}

// withHTTPPrincipal records the authenticated HTTP caller on the request.
func withHTTPPrincipal(principal string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), httpPrincipalKey{}, principal)))
	})
}

// callerPrincipal identifies a caller that AuthenticateRequest accepted: by
// its TLS client certificate, else by its bearer credential.  Only digests
// are kept, never the credential.
func callerPrincipal(r *http.Request) string {
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		sum := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
		return "tls:" + hex.EncodeToString(sum[:])
	}
	fields := strings.Fields(r.Header.Get("Authorization"))
	if len(fields) == 2 && strings.EqualFold(fields[0], "Bearer") {
		sum := sha256.Sum256([]byte(fields[1]))
		return "bearer:" + hex.EncodeToString(sum[:])
	}
	return "caller"
}
