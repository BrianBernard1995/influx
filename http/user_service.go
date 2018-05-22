package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/influxdata/platform"
	idpctx "github.com/influxdata/platform/context"
	"github.com/influxdata/platform/kit/errors"
	"github.com/julienschmidt/httprouter"
)

// UserHandler represents an HTTP API handler for users.
type UserHandler struct {
	*httprouter.Router

	UserService platform.UserService
}

// NewUserHandler returns a new instance of UserHandler.
func NewUserHandler() *UserHandler {
	h := &UserHandler{
		Router: httprouter.New(),
	}

	h.HandlerFunc("POST", "/v1/users", withAuth(h.handlePostUser, platform.CreateUserPermission))
	h.HandlerFunc("GET", "/v1/users", withAuth(h.handleGetUsers, platform.ReadUserPermission))
	h.HandlerFunc("GET", "/v1/users/:id", withAuth(h.handleGetUser, platform.ReadUserPermission))
	h.HandlerFunc("PATCH", "/v1/users/:id", withAuth(h.handlePatchUser, platform.WriteUserPermission))
	h.HandlerFunc("DELETE", "/v1/users/:id", withAuth(h.handleDeleteUser, platform.DeleteUserPermission))
	return h
}

func withAuth(next http.HandlerFunc, p platform.Permission) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth, err := idpctx.GetAuthorization(ctx)
		if err != nil {
			errors.EncodeHTTP(ctx, err, w)
			return
		}
		if !platform.Allowed(p, auth.Permissions) {
			http.Error(w, "you are not authorized for this action", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
		return
	}
}

// handlePostUser is the HTTP handler for the POST /v1/users route.
func (h *UserHandler) handlePostUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodePostUserRequest(ctx, r)
	if err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}

	if err := h.UserService.CreateUser(ctx, req.User); err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusCreated, req.User); err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}
}

type postUserRequest struct {
	User *platform.User
}

func decodePostUserRequest(ctx context.Context, r *http.Request) (*postUserRequest, error) {
	user := &platform.User{}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		return nil, err
	}

	return &postUserRequest{
		User: user,
	}, nil
}

// handleGetUser is the HTTP handler for the GET /v1/users/:id route.
func (h *UserHandler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodeGetUserRequest(ctx, r)
	if err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}

	user, err := h.UserService.FindUserByID(ctx, req.UserID)
	if err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusOK, user); err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}
}

type getUserRequest struct {
	UserID platform.ID
}

func decodeGetUserRequest(ctx context.Context, r *http.Request) (*getUserRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	idParam := params.ByName("id")
	if idParam == "" {
		return nil, errors.InvalidDataf("url missing id")
	}

	var id platform.ID
	if err := id.DecodeFromString(idParam); err != nil {
		return nil, err
	}

	req := &getUserRequest{
		UserID: id,
	}

	return req, nil
}

// handleDeleteUser is the HTTP handler for the DELETE /v1/users/:id route.
func (h *UserHandler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodeDeleteUserRequest(ctx, r)
	if err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}

	if err := h.UserService.DeleteUser(ctx, req.UserID); err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

type deleteUserRequest struct {
	UserID platform.ID
}

func decodeDeleteUserRequest(ctx context.Context, r *http.Request) (*deleteUserRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, errors.InvalidDataf("url missing id")
	}

	var i platform.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	return &deleteUserRequest{
		UserID: i,
	}, nil
}

// handleGetUsers is the HTTP handler for the GET /v1/users route.
func (h *UserHandler) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodeGetUsersRequest(ctx, r)
	if err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}

	users, _, err := h.UserService.FindUsers(ctx, req.filter)
	if err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusOK, users); err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}
}

type getUsersRequest struct {
	filter platform.UserFilter
}

func decodeGetUsersRequest(ctx context.Context, r *http.Request) (*getUsersRequest, error) {
	qp := r.URL.Query()
	req := &getUsersRequest{}

	if id := qp.Get("id"); id != "" {
		req.filter.ID = &platform.ID{}
		if err := req.filter.ID.DecodeFromString(id); err != nil {
			return nil, err
		}
	}

	if name := qp.Get("name"); name != "" {
		req.filter.Name = &name
	}

	return req, nil
}

// handlePatchUser is the HTTP handler for the PATCH /v1/users/:id route.
func (h *UserHandler) handlePatchUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodePatchUserRequest(ctx, r)
	if err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}

	user, err := h.UserService.UpdateUser(ctx, req.UserID, req.Update)
	if err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusOK, user); err != nil {
		errors.EncodeHTTP(ctx, err, w)
		return
	}
}

type patchUserRequest struct {
	Update platform.UserUpdate
	UserID platform.ID
}

func decodePatchUserRequest(ctx context.Context, r *http.Request) (*patchUserRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, errors.InvalidDataf("url missing id")
	}

	var i platform.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	var upd platform.UserUpdate
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		return nil, err
	}

	return &patchUserRequest{
		Update: upd,
		UserID: i,
	}, nil
}

// UserService connects to Influx via HTTP using tokens to manage users
type UserService struct {
	Addr               string
	Token              string
	InsecureSkipVerify bool
}

// FindUserByID returns a single user by ID.
func (s *UserService) FindUserByID(ctx context.Context, id platform.ID) (*platform.User, error) {
	url, err := newURL(s.Addr, userIDPath(id))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", s.Token)

	hc := newClient(url.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(resp.Header.Get("X-Influx-Error"))
	}

	var user platform.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &user, nil
}

// FindUser returns the first user that matches filter.
func (s *UserService) FindUser(ctx context.Context, filter platform.UserFilter) (*platform.User, error) {
	users, n, err := s.FindUsers(ctx, filter)
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, fmt.Errorf("found no matching user")
	}

	return users[0], nil
}

// FindUsers returns a list of users that match filter and the total count of matching users.
// Additional options provide pagination & sorting.
func (s *UserService) FindUsers(ctx context.Context, filter platform.UserFilter, opt ...platform.FindOptions) ([]*platform.User, int, error) {
	url, err := newURL(s.Addr, userPath)
	if err != nil {
		return nil, 0, err
	}

	query := url.Query()

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	if filter.ID != nil {
		query.Add("id", filter.ID.String())
	}
	if filter.Name != nil {
		query.Add("name", *filter.Name)
	}

	req.URL.RawQuery = query.Encode()
	req.Header.Set("Authorization", s.Token)

	hc := newClient(url.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf(resp.Header.Get("X-Influx-Error"))
	}

	var bs []*platform.User
	if err := json.NewDecoder(resp.Body).Decode(&bs); err != nil {
		return nil, 0, err
	}

	return bs, len(bs), nil
}

const (
	userPath = "/v1/users"
)

// CreateUser creates a new user and sets u.ID with the new identifier.
func (s *UserService) CreateUser(ctx context.Context, u *platform.User) error {
	url, err := newURL(s.Addr, userPath)
	if err != nil {
		return err
	}

	octets, err := json.Marshal(u)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url.String(), bytes.NewReader(octets))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", s.Token)

	hc := newClient(url.Scheme, s.InsecureSkipVerify)

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf(resp.Header.Get("X-Influx-Error"))
	}

	if err := json.NewDecoder(resp.Body).Decode(u); err != nil {
		return err
	}

	return nil
}

// UpdateUser updates a single user with changeset.
// Returns the new user state after update.
func (s *UserService) UpdateUser(ctx context.Context, id platform.ID, upd platform.UserUpdate) (*platform.User, error) {
	url, err := newURL(s.Addr, userIDPath(id))
	if err != nil {
		return nil, err
	}

	octets, err := json.Marshal(upd)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", url.String(), bytes.NewReader(octets))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", s.Token)

	hc := newClient(url.Scheme, s.InsecureSkipVerify)

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(resp.Header.Get("X-Influx-Error"))
	}

	var u platform.User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &u, nil
}

// DeleteUser removes a user by ID.
func (s *UserService) DeleteUser(ctx context.Context, id platform.ID) error {
	url, err := newURL(s.Addr, userIDPath(id))
	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", url.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", s.Token)

	hc := newClient(url.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf(resp.Header.Get("X-Influx-Error"))
	}

	return nil
}

func userIDPath(id platform.ID) string {
	return path.Join(userPath, id.String())
}
