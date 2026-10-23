package analytics

import (
	"net"
	"net/http"
	"net/netip"

	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
)

// Handlers exposes the service over HTTP.
type Handlers struct {
	svc *Service
	// Proxies allowed to tell us the visitor's country (CF-IPCountry).
	trusted []netip.Prefix
}

func NewHandlers(svc *Service, trustedProxies []netip.Prefix) *Handlers {
	return &Handlers{svc: svc, trusted: trustedProxies}
}

type pageViewInput struct {
	Path     string `json:"path" validate:"required,max=300"`
	Referrer string `json:"referrer" validate:"max=1000"`
}

// PageView handles POST /v1/events/pageview. It always answers 204 once the body is valid, so
// the beacon cannot be used to learn whether a request was counted.
func (h *Handlers) PageView(w http.ResponseWriter, r *http.Request) error {
	var in pageViewInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	ua := r.UserAgent()
	if len(ua) > 512 {
		ua = ua[:512]
	}
	country := ""
	if h.fromTrustedProxy(r) {
		country = r.Header.Get("CF-IPCountry")
	}
	err := h.svc.RecordPageView(r.Context(), PageView{
		Path:      in.Path,
		Referrer:  in.Referrer,
		IP:        httpx.ClientIPFrom(r.Context()),
		UserAgent: ua,
		Country:   country,
		OptOut:    r.Header.Get("DNT") == "1" || r.Header.Get("Sec-GPC") == "1",
	})
	if err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

// Report handles GET /v1/admin/analytics?days=7|30|90.
func (h *Handlers) Report(w http.ResponseWriter, r *http.Request) error {
	q := httpx.NewQuery(r)
	days := q.Enum("days", "30", "7", "30", "90")
	if err := q.Err(); err != nil {
		return err
	}
	n := map[string]int{"7": 7, "30": 30, "90": 90}[days]
	rep, err := h.svc.Report(r.Context(), n)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, rep)
	return nil
}

func (h *Handlers) fromTrustedProxy(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	ip = ip.Unmap()
	for _, p := range h.trusted {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}
