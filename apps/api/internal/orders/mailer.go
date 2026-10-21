package orders

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"

	"github.com/atcreations/atcreations/apps/api/internal/mail"
)

// Mailer renders and sends customer-facing order emails. html/template
// escapes every interpolated value, so customer-supplied names and addresses
// cannot inject markup.
type Mailer struct {
	sender        mail.Sender
	storeName     string
	storefrontURL string
}

func NewMailer(sender mail.Sender, storeName, storefrontURL string) *Mailer {
	return &Mailer{sender: sender, storeName: storeName, storefrontURL: storefrontURL}
}

func money(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	return fmt.Sprintf("%s$%d.%02d", sign, cents/100, cents%100)
}

var funcs = map[string]any{"money": money, "title": func(s Status) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(string(s[:1])) + string(s[1:])
}}

const layoutHTML = `<!doctype html><html><body style="font-family:Arial,sans-serif;color:#1f2937;max-width:600px;margin:auto">
<h2 style="color:#0071bc">{{.Store}}</h2>{{template "body" .}}
<p style="color:#6b7280;font-size:12px;margin-top:32px">This is an automated message from {{.Store}}. Please do not reply to this email.</p>
</body></html>`

var confirmationHTML = htmltemplate.Must(htmltemplate.Must(htmltemplate.New("layout").Funcs(funcs).Parse(layoutHTML)).New("body").Parse(`
<p>Hi {{.Order.Customer.Name}},</p>
<p>Thank you for your order <strong>{{.Order.Number}}</strong>. We will contact you shortly to arrange payment and delivery.</p>
<table style="width:100%;border-collapse:collapse">
{{range .Order.Items}}<tr><td style="padding:4px 0">{{.Name.EN}} × {{.Quantity}}</td><td style="text-align:right">{{money .LineTotalCents}}</td></tr>{{end}}
<tr><td style="padding-top:8px;border-top:1px solid #e5e7eb"><strong>Total</strong></td><td style="text-align:right;padding-top:8px;border-top:1px solid #e5e7eb"><strong>{{money .Order.TotalCents}}</strong></td></tr>
</table>
<p><strong>Shipping to:</strong><br>{{.Order.Shipping.Name}}<br>{{.Order.Shipping.Line1}}{{if .Order.Shipping.Line2}}<br>{{.Order.Shipping.Line2}}{{end}}<br>{{.Order.Shipping.City}} {{.Order.Shipping.Region}} {{.Order.Shipping.PostalCode}}<br>{{.Order.Shipping.Country}}</p>
<p>Your order access code is <strong style="font-family:monospace;font-size:16px">{{.AccessCode}}</strong>. Keep it private — you will need it with your email to check your order status.</p>
`))

var confirmationText = texttemplate.Must(texttemplate.New("t").Funcs(funcs).Parse(`Hi {{.Order.Customer.Name}},

Thank you for your order {{.Order.Number}}. We will contact you shortly to arrange payment and delivery.
{{range .Order.Items}}
- {{.Name.EN}} x {{.Quantity}}: {{money .LineTotalCents}}{{end}}

Total: {{money .Order.TotalCents}}

Order access code: {{.AccessCode}} (keep it private)

{{.Store}}
`))

var statusHTML = htmltemplate.Must(htmltemplate.Must(htmltemplate.New("layout").Funcs(funcs).Parse(layoutHTML)).New("body").Parse(`
<p>Hi {{.Order.Customer.Name}},</p>
<p>The status of your order <strong>{{.Order.Number}}</strong> is now <strong>{{title .Order.Status}}</strong>.</p>
{{with .Note}}<p>{{.}}</p>{{end}}
{{if .Order.TrackingNumber}}<p>Tracking number: <strong>{{.Order.TrackingNumber}}</strong></p>{{end}}
`))

var statusText = texttemplate.Must(texttemplate.New("t").Funcs(funcs).Parse(`Hi {{.Order.Customer.Name}},

The status of your order {{.Order.Number}} is now {{title .Order.Status}}.
{{with .Note}}
{{.}}
{{end}}{{if .Order.TrackingNumber}}
Tracking number: {{.Order.TrackingNumber}}
{{end}}
{{.Store}}
`))

type tmplData struct {
	Store      string
	Order      *Order
	AccessCode string
	Note       string
}

func render(h *htmltemplate.Template, t *texttemplate.Template, d tmplData) (string, string, error) {
	var hb, tb bytes.Buffer
	if err := h.ExecuteTemplate(&hb, "layout", d); err != nil {
		return "", "", err
	}
	if err := t.Execute(&tb, d); err != nil {
		return "", "", err
	}
	return hb.String(), tb.String(), nil
}

func (m *Mailer) Confirmation(ctx context.Context, o *Order, accessCode string) error {
	html, text, err := render(confirmationHTML, confirmationText, tmplData{Store: m.storeName, Order: o, AccessCode: accessCode})
	if err != nil {
		return err
	}
	return m.sender.Send(ctx, mail.Message{To: o.Customer.Email, ToName: o.Customer.Name,
		Subject: fmt.Sprintf("%s order %s received", m.storeName, o.Number), HTML: html, Text: text})
}

func (m *Mailer) StatusChanged(ctx context.Context, o *Order) error {
	note := ""
	if n := len(o.History); n > 0 {
		note = o.History[n-1].Note
	}
	html, text, err := render(statusHTML, statusText, tmplData{Store: m.storeName, Order: o, Note: note})
	if err != nil {
		return err
	}
	return m.sender.Send(ctx, mail.Message{To: o.Customer.Email, ToName: o.Customer.Name,
		Subject: fmt.Sprintf("%s order %s: %s", m.storeName, o.Number, o.Status), HTML: html, Text: text})
}
