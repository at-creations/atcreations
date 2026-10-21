package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRolePermissions(t *testing.T) {
	cases := []struct {
		role Role
		perm Permission
		want bool
	}{
		{RoleStaff, ProductsWrite, true},
		{RoleStaff, ProductsDelete, false},
		{RoleStaff, CategoriesWrite, false},
		{RoleStaff, StaffManage, false},
		{RoleManager, CategoriesWrite, true},
		{RoleManager, OrdersShipping, true},
		{RoleManager, StoreWrite, false},
		{RoleManager, StaffManage, false},
		{RoleAdmin, StaffManage, true},
		{RoleAdmin, AuditRead, true},
		{Role("superuser"), ProductsWrite, false},
		{Role(""), OrdersRead, false},
	}
	for _, c := range cases {
		if got := c.role.Can(c.perm); got != c.want {
			t.Errorf("%q.Can(%s) = %v, want %v", c.role, c.perm, got, c.want)
		}
	}
}

func TestRequire(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := Require(StaffManage)(ok)

	for _, c := range []struct {
		p    *Principal
		want int
	}{
		{nil, http.StatusForbidden},
		{&Principal{UserID: "u", Role: RoleManager}, http.StatusForbidden},
		{&Principal{UserID: "u", Role: RoleAdmin}, http.StatusNoContent},
	} {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		if c.p != nil {
			req = req.WithContext(WithPrincipal(req.Context(), c.p))
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("principal %+v: got %d, want %d", c.p, rec.Code, c.want)
		}
	}
}

func TestPermissionsOfReturnsCopy(t *testing.T) {
	p := PermissionsOf(RoleStaff)
	p[0] = StaffManage
	if RoleStaff.Can(StaffManage) {
		t.Fatal("mutating the returned slice changed role permissions")
	}
}
