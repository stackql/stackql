package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/public/secrets"
	lrucache "github.com/stackql/stackql-parser/go/cache"
	"github.com/stackql/stackql/internal/stackql/jsonpath"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
)

type providerSessionState struct {
	variables    map[string]any
	base         omnisdk.Args
	mu           sync.Mutex
	client       omnisdk.ClientSession
	store        secrets.Store
	bindings     map[string]omnisdk.ProviderBinding
	documentView string
}

func decodeSessionObject(value interface{}, target interface{}) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("invalid session configuration")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid session configuration field or type")
	}
	return nil
}

func updateSessionField(fields map[string]interface{}, parts []string, rhs interface{}) error {
	for _, part := range parts[:len(parts)-1] {
		child, ok := fields[part].(map[string]interface{})
		if !ok {
			return fmt.Errorf("session configuration path does not identify an object")
		}
		fields = child
	}
	key := parts[len(parts)-1]
	if rhs == nil {
		delete(fields, key)
	} else {
		fields[key] = rhs
	}
	return nil
}

func (hc *standardHandlerContext) providerSessionState() *providerSessionState {
	hc.sessionCtxMutex.Lock()
	defer hc.sessionCtxMutex.Unlock()
	if hc.providerSession == nil {
		hc.providerSession = newProviderSessionState(hc.runtimeContext.APIRequestTimeout, hc.runtimeContext.HTTPPageLimit)
	}
	return hc.providerSession
}

func newProviderSessionState(seconds int, pageLimit int) *providerSessionState {
	timeout := time.Duration(seconds) * time.Second
	if seconds == 0 {
		timeout = -1
	}
	return &providerSessionState{
		base:  omnisdk.Args{Tuning: omnisdk.Tuning{RequestTimeout: timeout, PageLimit: pageLimit}},
		store: newSessionSecretStore(),
	}
}

func (state *providerSessionState) sessionArgs() omnisdk.Args {
	args := state.base
	args.ProviderAliases = state.bindings
	return args
}

func (state *providerSessionState) initialize() error {
	if state.client != nil {
		return nil
	}
	client, err := omnisdk.NewClientSession(state.sessionArgs(), state.store)
	if err != nil {
		return err
	}
	state.client = client
	return nil
}

//nolint:mnd // One canonical and one SQL-prefixed entry per alias.
func (hc *standardHandlerContext) ProviderSessionArgs(ctx context.Context) (omnisdk.Args, error) {
	state := hc.providerSessionState()
	state.mu.Lock()
	if err := state.initialize(); err != nil {
		state.mu.Unlock()
		return omnisdk.Args{}, err
	}
	client := state.client
	state.mu.Unlock()
	args, err := client.Snapshot(ctx)
	if err != nil {
		return omnisdk.Args{}, err
	}
	aliases := make(map[string]omnisdk.ProviderBinding, len(args.ProviderAliases)*2)
	for alias, binding := range args.ProviderAliases {
		aliases[alias] = binding
		aliases["stackql_unstable_"+alias] = binding
	}
	args.ProviderAliases = aliases
	return args, nil
}

func (hc *standardHandlerContext) ResolveProviderAddress(address string) string {
	state := hc.providerSessionState()
	state.mu.Lock()
	defer state.mu.Unlock()
	provider, rest, _ := strings.Cut(address, ".")
	provider = strings.TrimPrefix(provider, "stackql_unstable_")
	if binding, ok := state.bindings[provider]; ok {
		if rest == "" {
			return binding.Provider
		}
		return "stackql_unstable_" + strings.TrimPrefix(binding.Provider, "stackql_unstable_") + "." + rest
	}
	return address
}

//nolint:gocognit,gocyclo,cyclop,govet,mnd,gosec,nestif,funlen // Explicit configuration and trusted credential storage.
func (hc *standardHandlerContext) setProviderSessionConfig(path string, rhs interface{}, scope string) (bool, error) {
	parts, err := jsonpath.SplitSearchPath(path)
	if err != nil {
		return true, err
	}
	if len(parts) == 0 {
		return false, nil
	}
	switch parts[0] {
	case "providers", "secret_store", "secrets":
	default:
		return false, nil
	}
	if scope != "" && scope != "session" {
		return true, fmt.Errorf("provider and secret configuration requires session scope")
	}
	state := hc.providerSessionState()
	state.mu.Lock()
	defer state.mu.Unlock()
	switch parts[0] {
	case "secret_store":
		if len(parts) == 1 && rhs == nil {
			state.store, state.client = newSessionSecretStore(), nil
			return true, nil
		}
		if !hc.runtimeContext.AllowPersistentSecrets {
			return true, fmt.Errorf("persistent secret stores are disabled; operator must enable --allow-persistent-secrets")
		}
		if len(parts) != 1 || rhs == nil {
			return true, fmt.Errorf("secret_store requires an explicit configuration object")
		}
		var cfg secrets.Config
		if err := decodeSessionObject(rhs, &cfg); err != nil {
			return true, err
		}
		if cfg.Backend != "os" || cfg.Path != "" {
			return true, fmt.Errorf("explicit OS backend required; flat-file secrets are not supported")
		}
		store, err := secrets.New(cfg)
		if err != nil {
			return true, err
		}
		client, err := omnisdk.NewClientSession(state.sessionArgs(), store)
		if err != nil {
			return true, err
		}
		state.store, state.client = store, client
		return true, nil
	case "secrets":
		if len(parts) != 2 || parts[1] == "" {
			return true, fmt.Errorf("secrets requires one explicit key")
		}
		if state.store == nil {
			return true, fmt.Errorf("configure secret_store first")
		}
		store := state.store
		state.mu.Unlock()
		defer state.mu.Lock()
		if rhs == nil {
			return true, store.Delete(hc.GetExecutionContext(), parts[1])
		}
		value, ok := rhs.(string)
		if !ok {
			return true, fmt.Errorf("secret value must be a string")
		}
		return true, store.Set(hc.GetExecutionContext(), parts[1], []byte(value))
	default:
		if len(parts) < 2 {
			return true, fmt.Errorf("providers requires an alias")
		}
		if err := state.initialize(); err != nil {
			return true, err
		}
		alias := parts[1]
		if strings.HasPrefix(alias, "stackql_unstable_") {
			return true, fmt.Errorf("alias must not use the reserved provider prefix")
		}
		if len(parts) == 2 && rhs == nil {
			if err := state.client.UnsetProvider(alias); err != nil {
				return true, err
			}
			delete(state.bindings, alias)
			return true, nil
		}
		fields := map[string]interface{}{}
		if len(parts) == 2 {
			var ok bool
			fields, ok = rhs.(map[string]interface{})
			if !ok {
				return true, fmt.Errorf("provider binding must be an object")
			}
		} else {
			binding, ok := state.bindings[alias]
			if !ok {
				return true, fmt.Errorf("unknown provider alias")
			}
			raw, _ := json.Marshal(binding)
			if err := json.Unmarshal(raw, &fields); err != nil {
				return true, err
			}
			if err := updateSessionField(fields, parts[2:], rhs); err != nil {
				return true, err
			}
		}
		var binding omnisdk.ProviderBinding
		if err := decodeSessionObject(fields, &binding); err != nil {
			return true, err
		}
		if err := state.client.SetProvider(alias, binding); err != nil {
			return true, err
		}
		if state.bindings == nil {
			state.bindings = map[string]omnisdk.ProviderBinding{}
		}
		state.bindings[alias] = binding
		return true, nil
	}
}

func (hc *standardHandlerContext) ReplaceAuthContext(provider string, auth *dto.AuthCtx) {
	hc.authMapMutex.Lock()
	defer hc.authMapMutex.Unlock()
	if provider == "" {
		provider = hc.runtimeContext.ProviderStr
	}
	if hc.authContexts == nil {
		hc.authContexts = dto.AuthContexts{}
	}
	hc.authContexts[provider] = auth.Clone()
	delete(hc.docAuthProviders, provider)
}

func (hc *standardHandlerContext) ProviderBindingParams(address string) map[string]string {
	state := hc.providerSessionState()
	state.mu.Lock()
	defer state.mu.Unlock()
	provider, _, _ := strings.Cut(address, ".")
	provider = strings.TrimPrefix(provider, "stackql_unstable_")
	params := map[string]string{}
	for key, value := range state.bindings[provider].Params {
		params[key] = value
	}
	return params
}

func (source *standardHandlerContext) NewClientContext() HandlerContext {
	// This concrete receiver's Clone always returns its own type.
	target := source.Clone().(*standardHandlerContext) //nolint:errcheck
	// Plans capture session configuration; client caches must not cross session boundaries.
	target.lRUCache = lrucache.NewLRUCache(int64(source.runtimeContext.QueryCacheSize))
	// Provider registration installs auth defaults, so both caches need the same client lifetime.
	source.providersMapMutex.Lock()
	target.providers = maps.Clone(source.providers)
	target.providersMapMutex = &sync.Mutex{}
	source.authMapMutex.Lock()
	target.authContexts = source.authContexts.Clone()
	target.docAuthProviders = map[string]struct{}{}
	for name := range source.docAuthProviders {
		target.docAuthProviders[name] = struct{}{}
	}
	source.authMapMutex.Unlock()
	source.providersMapMutex.Unlock()
	target.authMapMutex = &sync.Mutex{}
	target.sessionCtxMutex = &sync.Mutex{}
	state := source.providerSessionState()
	state.mu.Lock()
	target.providerSession = newProviderSessionState(
		source.runtimeContext.APIRequestTimeout, source.runtimeContext.HTTPPageLimit,
	)
	target.providerSession.variables = maps.Clone(state.variables)
	target.providerSession.documentView = state.documentView
	target.providerSession.bindings = make(map[string]omnisdk.ProviderBinding, len(state.bindings))
	for alias, binding := range state.bindings {
		binding.Params = maps.Clone(binding.Params)
		binding.Auth = cloneSessionAuth(binding.Auth)
		target.providerSession.bindings[alias] = binding
	}
	state.mu.Unlock()
	return target
}

func cloneSessionAuth(auth *omnisdk.Auth) *omnisdk.Auth {
	if auth == nil {
		return nil
	}
	cloned := *auth
	cloned.Scopes = slices.Clone(auth.Scopes)
	cloned.Successor = cloneSessionAuth(auth.Successor)
	if auth.TLS != nil {
		tls := *auth.TLS
		cloned.TLS = &tls
	}
	return &cloned
}
