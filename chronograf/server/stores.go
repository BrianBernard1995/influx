package server

import (
	"context"

	"github.com/EMCECS/influx/chronograf"
	"github.com/EMCECS/influx/chronograf/noop"
	"github.com/EMCECS/influx/chronograf/organizations"
	"github.com/EMCECS/influx/chronograf/roles"
)

// hasOrganizationContext retrieves organization specified on context
// under the organizations.ContextKey
func hasOrganizationContext(ctx context.Context) (string, bool) {
	// prevents panic in case of nil context
	if ctx == nil {
		return "", false
	}
	orgID, ok := ctx.Value(organizations.ContextKey).(string)
	// should never happen
	if !ok {
		return "", false
	}
	if orgID == "" {
		return "", false
	}
	return orgID, true
}

// hasRoleContext retrieves organization specified on context
// under the organizations.ContextKey
func hasRoleContext(ctx context.Context) (string, bool) {
	// prevents panic in case of nil context
	if ctx == nil {
		return "", false
	}
	role, ok := ctx.Value(roles.ContextKey).(string)
	// should never happen
	if !ok {
		return "", false
	}
	switch role {
	case roles.MemberRoleName, roles.ViewerRoleName, roles.EditorRoleName, roles.AdminRoleName:
		return role, true
	default:
		return "", false
	}
}

type userContextKey string

// UserContextKey is the context key for retrieving the user off of context
const UserContextKey = userContextKey("user")

// hasUserContext speficies if the context contains
// the UserContextKey and that the value stored there is chronograf.User
func hasUserContext(ctx context.Context) (*chronograf.User, bool) {
	// prevents panic in case of nil context
	if ctx == nil {
		return nil, false
	}
	u, ok := ctx.Value(UserContextKey).(*chronograf.User)
	// should never happen
	if !ok {
		return nil, false
	}
	if u == nil {
		return nil, false
	}
	return u, true
}

// hasSuperAdminContext speficies if the context contains
// the UserContextKey user is a super admin
func hasSuperAdminContext(ctx context.Context) bool {
	u, ok := hasUserContext(ctx)
	if !ok {
		return false
	}
	return u.SuperAdmin
}

// DataStore is collection of resources that are used by the Service
// Abstracting this into an interface was useful for isolated testing
type DataStore interface {
	Sources(ctx context.Context) chronograf.SourcesStore
	Servers(ctx context.Context) chronograf.ServersStore
	Layouts(ctx context.Context) chronograf.LayoutsStore
	Users(ctx context.Context) chronograf.UsersStore
	Organizations(ctx context.Context) chronograf.OrganizationsStore
	Mappings(ctx context.Context) chronograf.MappingsStore
	Dashboards(ctx context.Context) chronograf.DashboardsStore
	Config(ctx context.Context) chronograf.ConfigStore
	OrganizationConfig(ctx context.Context) chronograf.OrganizationConfigStore
}

// ensure that Store implements a DataStore
var _ DataStore = &Store{}

// Store implements the DataStore interface
type Store struct {
	SourcesStore            chronograf.SourcesStore
	ServersStore            chronograf.ServersStore
	LayoutsStore            chronograf.LayoutsStore
	UsersStore              chronograf.UsersStore
	DashboardsStore         chronograf.DashboardsStore
	MappingsStore           chronograf.MappingsStore
	OrganizationsStore      chronograf.OrganizationsStore
	ConfigStore             chronograf.ConfigStore
	OrganizationConfigStore chronograf.OrganizationConfigStore
}

// Sources returns a noop.SourcesStore if the context has no organization specified
// and an organization.SourcesStore otherwise.
func (s *Store) Sources(ctx context.Context) chronograf.SourcesStore {
	if isServer := hasServerContext(ctx); isServer {
		return s.SourcesStore
	}
	if org, ok := hasOrganizationContext(ctx); ok {
		return organizations.NewSourcesStore(s.SourcesStore, org)
	}

	return &noop.SourcesStore{}
}

// Servers returns a noop.ServersStore if the context has no organization specified
// and an organization.ServersStore otherwise.
func (s *Store) Servers(ctx context.Context) chronograf.ServersStore {
	if isServer := hasServerContext(ctx); isServer {
		return s.ServersStore
	}
	if org, ok := hasOrganizationContext(ctx); ok {
		return organizations.NewServersStore(s.ServersStore, org)
	}

	return &noop.ServersStore{}
}

// Layouts returns all layouts in the underlying layouts store.
func (s *Store) Layouts(ctx context.Context) chronograf.LayoutsStore {
	return s.LayoutsStore
}

// Users returns a chronograf.UsersStore.
// If the context is a server context, then the underlying chronograf.UsersStore
// is returned.
// If there is an organization specified on context, then an organizations.UsersStore
// is returned.
// If niether are specified, a noop.UsersStore is returned.
func (s *Store) Users(ctx context.Context) chronograf.UsersStore {
	if isServer := hasServerContext(ctx); isServer {
		return s.UsersStore
	}
	if org, ok := hasOrganizationContext(ctx); ok {
		return organizations.NewUsersStore(s.UsersStore, org)
	}

	return &noop.UsersStore{}
}

// Dashboards returns a noop.DashboardsStore if the context has no organization specified
// and an organization.DashboardsStore otherwise.
func (s *Store) Dashboards(ctx context.Context) chronograf.DashboardsStore {
	if isServer := hasServerContext(ctx); isServer {
		return s.DashboardsStore
	}
	if org, ok := hasOrganizationContext(ctx); ok {
		return organizations.NewDashboardsStore(s.DashboardsStore, org)
	}

	return &noop.DashboardsStore{}
}

// OrganizationConfig returns a noop.OrganizationConfigStore if the context has no organization specified
// and an organization.OrganizationConfigStore otherwise.
func (s *Store) OrganizationConfig(ctx context.Context) chronograf.OrganizationConfigStore {
	if orgID, ok := hasOrganizationContext(ctx); ok {
		return organizations.NewOrganizationConfigStore(s.OrganizationConfigStore, orgID)
	}

	return &noop.OrganizationConfigStore{}
}

// Organizations returns the underlying OrganizationsStore.
func (s *Store) Organizations(ctx context.Context) chronograf.OrganizationsStore {
	if isServer := hasServerContext(ctx); isServer {
		return s.OrganizationsStore
	}
	if isSuperAdmin := hasSuperAdminContext(ctx); isSuperAdmin {
		return s.OrganizationsStore
	}
	if org, ok := hasOrganizationContext(ctx); ok {
		return organizations.NewOrganizationsStore(s.OrganizationsStore, org)
	}
	return &noop.OrganizationsStore{}
}

// Config returns the underlying ConfigStore.
func (s *Store) Config(ctx context.Context) chronograf.ConfigStore {
	if isServer := hasServerContext(ctx); isServer {
		return s.ConfigStore
	}
	if isSuperAdmin := hasSuperAdminContext(ctx); isSuperAdmin {
		return s.ConfigStore
	}

	return &noop.ConfigStore{}
}

// Mappings returns the underlying MappingsStore.
func (s *Store) Mappings(ctx context.Context) chronograf.MappingsStore {
	if isServer := hasServerContext(ctx); isServer {
		return s.MappingsStore
	}
	if isSuperAdmin := hasSuperAdminContext(ctx); isSuperAdmin {
		return s.MappingsStore
	}
	return &noop.MappingsStore{}
}

// ensure that DirectStore implements a DataStore
var _ DataStore = &DirectStore{}

// Store implements the DataStore interface
type DirectStore struct {
	SourcesStore            chronograf.SourcesStore
	ServersStore            chronograf.ServersStore
	LayoutsStore            chronograf.LayoutsStore
	UsersStore              chronograf.UsersStore
	DashboardsStore         chronograf.DashboardsStore
	MappingsStore           chronograf.MappingsStore
	OrganizationsStore      chronograf.OrganizationsStore
	ConfigStore             chronograf.ConfigStore
	OrganizationConfigStore chronograf.OrganizationConfigStore
}

// Sources returns a noop.SourcesStore if the context has no organization specified
// and an organization.SourcesStore otherwise.
func (s *DirectStore) Sources(ctx context.Context) chronograf.SourcesStore {
	return s.SourcesStore
}

// Servers returns a noop.ServersStore if the context has no organization specified
// and an organization.ServersStore otherwise.
func (s *DirectStore) Servers(ctx context.Context) chronograf.ServersStore {
	return s.ServersStore
}

// Layouts returns all layouts in the underlying layouts store.
func (s *DirectStore) Layouts(ctx context.Context) chronograf.LayoutsStore {
	return s.LayoutsStore
}

// Users returns a chronograf.UsersStore.
// If the context is a server context, then the underlying chronograf.UsersStore
// is returned.
// If there is an organization specified on context, then an organizations.UsersStore
// is returned.
// If niether are specified, a noop.UsersStore is returned.
func (s *DirectStore) Users(ctx context.Context) chronograf.UsersStore {
	return s.UsersStore
}

// Dashboards returns a noop.DashboardsStore if the context has no organization specified
// and an organization.DashboardsStore otherwise.
func (s *DirectStore) Dashboards(ctx context.Context) chronograf.DashboardsStore {
	return s.DashboardsStore
}

// OrganizationConfig returns a noop.OrganizationConfigStore if the context has no organization specified
// and an organization.OrganizationConfigStore otherwise.
func (s *DirectStore) OrganizationConfig(ctx context.Context) chronograf.OrganizationConfigStore {
	return s.OrganizationConfigStore
}

// Organizations returns the underlying OrganizationsStore.
func (s *DirectStore) Organizations(ctx context.Context) chronograf.OrganizationsStore {
	return s.OrganizationsStore
}

// Config returns the underlying ConfigStore.
func (s *DirectStore) Config(ctx context.Context) chronograf.ConfigStore {
	return s.ConfigStore
}

// Mappings returns the underlying MappingsStore.
func (s *DirectStore) Mappings(ctx context.Context) chronograf.MappingsStore {
	return s.MappingsStore
}
