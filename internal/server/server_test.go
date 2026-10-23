package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"

	"icecream-api/internal/auth"
	"icecream-api/internal/database"
	"icecream-api/internal/models"
	"icecream-api/internal/server"
)

type creds struct{ email, password string }

var (
	owner   = creds{"owner@icecream.test", "Owner@123"}
	manager = creds{"manager@icecream.test", "Manager@123"}
	cashier = creds{"cashier@icecream.test", "Cashier@123"}
)

// dbName extracts the database name from user:pass@tcp(host:port)/dbname?params
func dbName(dsn string) string {
	if i := strings.LastIndex(dsn, "/"); i >= 0 {
		dsn = dsn[i+1:]
	}
	if i := strings.Index(dsn, "?"); i >= 0 {
		dsn = dsn[:i]
	}
	return dsn
}

func setup(t *testing.T) *fiber.App {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set; skipping MySQL integration tests (see README)")
	}
	// the tests DROP all tables, so refuse to run against anything that isn't clearly a test DB
	if !strings.Contains(strings.ToLower(dbName(dsn)), "test") {
		t.Fatal("TEST_DATABASE_DSN must point to a database whose name contains \"test\"")
	}
	db, err := database.Connect(dsn)
	if err != nil {
		t.Fatal(err)
	}
	// start every test from an empty schema (children first because of the FK)
	if err := db.Migrator().DropTable(&models.Item{}, &models.Category{}, &models.Customer{}, &models.RevokedToken{}, &models.User{}); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := database.Seed(db, bcrypt.MinCost); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return server.New(db, auth.NewTokenManager("test-secret-test-secret", time.Hour))
}

// call performs a request and returns the status and decoded JSON body (if any).
func call(t *testing.T, app *fiber.App, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("response is not JSON: %s", raw)
		}
	}
	return resp.StatusCode, out
}

func login(t *testing.T, app *fiber.App, c creds) string {
	t.Helper()
	status, res := call(t, app, http.MethodPost, "/api/auth/login", "", map[string]string{"email": c.email, "password": c.password})
	if status != http.StatusOK {
		t.Fatalf("login %s: status %d body %v", c.email, status, res)
	}
	return res["data"].(map[string]any)["token"].(string)
}

func idOf(res map[string]any) int { return int(res["data"].(map[string]any)["id"].(float64)) }

func newItem(name string) map[string]any {
	return map[string]any{"name": name, "description": "test", "category_id": 1, "price": 3.5, "quantity": 10}
}

func newCustomer(email string) map[string]any {
	return map[string]any{"name": "Test Person", "email": email, "phone": "0771234567", "address": "Somewhere"}
}

func TestLogin(t *testing.T) {
	app := setup(t)

	status, res := call(t, app, http.MethodPost, "/api/auth/login", "", map[string]string{"email": owner.email, "password": owner.password})
	if status != 200 {
		t.Fatalf("want 200 got %d", status)
	}
	data := res["data"].(map[string]any)
	if data["token"] == "" || data["user"].(map[string]any)["role"] != "owner" {
		t.Fatalf("unexpected login response: %v", res)
	}
	if _, leaked := data["user"].(map[string]any)["PasswordHash"]; leaked {
		t.Fatal("password hash must never be serialised")
	}

	if s, _ := call(t, app, http.MethodPost, "/api/auth/login", "", map[string]string{"email": owner.email, "password": "wrong"}); s != 401 {
		t.Fatalf("wrong password: want 401 got %d", s)
	}
	if s, _ := call(t, app, http.MethodPost, "/api/auth/login", "", map[string]string{"email": "nobody@x.com", "password": "x"}); s != 401 {
		t.Fatalf("unknown user: want 401 got %d", s)
	}
	if s, _ := call(t, app, http.MethodPost, "/api/auth/login", "", map[string]string{"email": "not-an-email"}); s != 422 {
		t.Fatalf("invalid payload: want 422 got %d", s)
	}
}

func TestProtectedRoutesRequireAuth(t *testing.T) {
	app := setup(t)
	for _, p := range []string{"/api/items", "/api/items/1", "/api/customers", "/api/customers/1", "/api/auth/me"} {
		if s, _ := call(t, app, http.MethodGet, p, "", nil); s != 401 {
			t.Errorf("GET %s without token: want 401 got %d", p, s)
		}
		if s, _ := call(t, app, http.MethodGet, p, "not.a.token", nil); s != 401 {
			t.Errorf("GET %s with bad token: want 401 got %d", p, s)
		}
	}
}

func TestLogoutRevokesToken(t *testing.T) {
	app := setup(t)
	tok := login(t, app, cashier)

	if s, _ := call(t, app, http.MethodGet, "/api/items", tok, nil); s != 200 {
		t.Fatalf("want 200 got %d", s)
	}
	if s, _ := call(t, app, http.MethodPost, "/api/auth/logout", tok, nil); s != 200 {
		t.Fatalf("logout: want 200 got %d", s)
	}
	if s, _ := call(t, app, http.MethodGet, "/api/items", tok, nil); s != 401 {
		t.Fatalf("revoked token should be rejected, got %d", s)
	}
}

// Owner: full | Manager: add/view/edit, no delete | Cashier: view/edit only for items
func TestItemPermissionMatrix(t *testing.T) {
	app := setup(t)
	ownerTok := login(t, app, owner)

	cases := []struct {
		name                       string
		who                        creds
		view, create, edit, delete int
	}{
		{"owner", owner, 200, 201, 200, 204},
		{"manager", manager, 200, 201, 200, 403},
		{"cashier", cashier, 200, 403, 200, 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tok := login(t, app, tc.who)

			if s, _ := call(t, app, http.MethodGet, "/api/items", tok, nil); s != tc.view {
				t.Errorf("list: want %d got %d", tc.view, s)
			}
			if s, _ := call(t, app, http.MethodPost, "/api/items", tok, newItem("Perm Test "+tc.name)); s != tc.create {
				t.Errorf("create: want %d got %d", tc.create, s)
			}

			// an item created by the owner to edit/delete
			_, created := call(t, app, http.MethodPost, "/api/items", ownerTok, newItem("Target "+tc.name))
			path := fmt.Sprintf("/api/items/%d", idOf(created))

			if s, _ := call(t, app, http.MethodPatch, path, tok, map[string]any{"quantity": 99}); s != tc.edit {
				t.Errorf("edit: want %d got %d", tc.edit, s)
			}
			if s, _ := call(t, app, http.MethodDelete, path, tok, nil); s != tc.delete {
				t.Errorf("delete: want %d got %d", tc.delete, s)
			}
		})
	}
}

// Owner: full | Manager: add/view/edit, no delete | Cashier: add/view/edit, no delete
func TestCustomerPermissionMatrix(t *testing.T) {
	app := setup(t)
	ownerTok := login(t, app, owner)

	cases := []struct {
		name                       string
		who                        creds
		view, create, edit, delete int
	}{
		{"owner", owner, 200, 201, 200, 204},
		{"manager", manager, 200, 201, 200, 403},
		{"cashier", cashier, 200, 201, 200, 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tok := login(t, app, tc.who)

			if s, _ := call(t, app, http.MethodGet, "/api/customers", tok, nil); s != tc.view {
				t.Errorf("list: want %d got %d", tc.view, s)
			}
			if s, _ := call(t, app, http.MethodPost, "/api/customers", tok, newCustomer(tc.name+"-own@example.com")); s != tc.create {
				t.Errorf("create: want %d got %d", tc.create, s)
			}

			_, created := call(t, app, http.MethodPost, "/api/customers", ownerTok, newCustomer(tc.name+"-target@example.com"))
			path := fmt.Sprintf("/api/customers/%d", idOf(created))

			if s, _ := call(t, app, http.MethodPut, path, tok, map[string]any{"name": "Renamed"}); s != tc.edit {
				t.Errorf("edit: want %d got %d", tc.edit, s)
			}
			if s, _ := call(t, app, http.MethodDelete, path, tok, nil); s != tc.delete {
				t.Errorf("delete: want %d got %d", tc.delete, s)
			}
		})
	}
}

func TestItemCRUDSearchAndValidation(t *testing.T) {
	app := setup(t)
	tok := login(t, app, owner)

	// create
	status, res := call(t, app, http.MethodPost, "/api/items", tok, newItem("Mint Chocolate Chip"))
	if status != 201 {
		t.Fatalf("create: %d %v", status, res)
	}
	id := idOf(res)
	path := fmt.Sprintf("/api/items/%d", id)
	if res["data"].(map[string]any)["category"].(map[string]any)["name"] != "Chocolate" {
		t.Fatalf("category should be embedded: %v", res)
	}

	// read
	if s, _ := call(t, app, http.MethodGet, path, tok, nil); s != 200 {
		t.Fatalf("get: %d", s)
	}

	// partial update keeps untouched fields
	status, res = call(t, app, http.MethodPatch, path, tok, map[string]any{"price": 4.25})
	data := res["data"].(map[string]any)
	if status != 200 || data["price"].(float64) != 4.25 || data["name"] != "Mint Chocolate Chip" {
		t.Fatalf("patch: %d %v", status, res)
	}

	// search is case-insensitive and partial
	_, res = call(t, app, http.MethodGet, "/api/items?search=MINT", tok, nil)
	if got := len(res["data"].([]any)); got != 1 {
		t.Fatalf("search: want 1 result got %d", got)
	}
	_, res = call(t, app, http.MethodGet, "/api/items?search=zzzz", tok, nil)
	if got := len(res["data"].([]any)); got != 0 {
		t.Fatalf("search: want 0 results got %d", got)
	}

	// filter by category + pagination meta
	_, res = call(t, app, http.MethodGet, "/api/items?category_id=2&limit=1", tok, nil)
	meta := res["meta"].(map[string]any)
	if len(res["data"].([]any)) != 1 || meta["total"].(float64) != 2 || meta["total_pages"].(float64) != 2 {
		t.Fatalf("filter/pagination: %v", res)
	}

	// validation
	bad := []map[string]any{
		{"name": "", "category_id": 1, "price": 1, "quantity": 1},
		{"name": "X", "category_id": 1, "price": -1, "quantity": 1},
		{"name": "X", "category_id": 1, "price": 1, "quantity": -5},
		{"name": "X", "category_id": 999, "price": 1, "quantity": 1},
		{"name": "X", "price": 1, "quantity": 1},
	}
	for i, b := range bad {
		if s, _ := call(t, app, http.MethodPost, "/api/items", tok, b); s != 422 {
			t.Errorf("bad payload %d: want 422 got %d", i, s)
		}
	}
	if s, _ := call(t, app, http.MethodPatch, path, tok, map[string]any{"price": -2}); s != 422 {
		t.Errorf("patch with negative price: want 422 got %d", s)
	}
	if s, _ := call(t, app, http.MethodGet, "/api/items/abc", tok, nil); s != 400 {
		t.Errorf("non-numeric id: want 400 got %d", s)
	}

	// delete + 404
	if s, _ := call(t, app, http.MethodDelete, path, tok, nil); s != 204 {
		t.Fatalf("delete: %d", s)
	}
	if s, _ := call(t, app, http.MethodGet, path, tok, nil); s != 404 {
		t.Fatalf("get after delete: want 404 got %d", s)
	}
	if s, _ := call(t, app, http.MethodDelete, path, tok, nil); s != 404 {
		t.Fatalf("delete twice: want 404 got %d", s)
	}
}

func TestCustomerCRUDSearchAndUniqueEmail(t *testing.T) {
	app := setup(t)
	tok := login(t, app, manager)

	status, res := call(t, app, http.MethodPost, "/api/customers", tok, newCustomer("Jane.Doe@Example.com"))
	if status != 201 {
		t.Fatalf("create: %d %v", status, res)
	}
	if res["data"].(map[string]any)["email"] != "jane.doe@example.com" {
		t.Fatalf("email should be normalised to lowercase: %v", res)
	}
	id := idOf(res)

	// duplicate email -> 409 (also case-insensitively)
	if s, _ := call(t, app, http.MethodPost, "/api/customers", tok, newCustomer("JANE.DOE@example.com")); s != 409 {
		t.Fatalf("duplicate create: want 409 got %d", s)
	}
	// updating to another customer's email -> 409
	if s, _ := call(t, app, http.MethodPatch, fmt.Sprintf("/api/customers/%d", id), tok, map[string]any{"email": "nimal@example.com"}); s != 409 {
		t.Fatalf("duplicate update: want 409 got %d", s)
	}
	// keeping own email on update is fine
	if s, _ := call(t, app, http.MethodPatch, fmt.Sprintf("/api/customers/%d", id), tok, map[string]any{"email": "jane.doe@example.com", "name": "Jane D"}); s != 200 {
		t.Fatalf("update own email: want 200 got %d", s)
	}

	// validation
	if s, _ := call(t, app, http.MethodPost, "/api/customers", tok, map[string]any{"name": "A", "email": "bad", "phone": "1"}); s != 422 {
		t.Fatalf("invalid customer: want 422 got %d", s)
	}

	// search by name / email / phone
	for _, q := range []string{"jane", "doe@example", "0771234"} {
		_, res = call(t, app, http.MethodGet, "/api/customers?search="+q, tok, nil)
		if len(res["data"].([]any)) < 1 {
			t.Errorf("search %q returned nothing", q)
		}
	}
	if s, _ := call(t, app, http.MethodGet, "/api/customers/99999", tok, nil); s != 404 {
		t.Fatalf("missing customer: want 404 got %d", s)
	}
}

func TestCategoriesEndpoint(t *testing.T) {
	app := setup(t)
	tok := login(t, app, cashier)
	status, res := call(t, app, http.MethodGet, "/api/categories", tok, nil)
	if status != 200 || len(res["data"].([]any)) != 3 {
		t.Fatalf("categories: %d %v", status, res)
	}
}
