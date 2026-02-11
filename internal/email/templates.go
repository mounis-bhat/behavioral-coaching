package email

import (
	"html"
	"strings"
)

const (
	appName    = "Starter"
	brandColor = "#4f46e5"
	bgColor    = "#f4f4f5"
)

type EmailParams struct {
	Greeting   string
	BodyLines  []string
	ButtonText string
	ButtonURL  string
	FooterText string
}

func RenderHTML(p EmailParams) string {
	var b strings.Builder

	b.WriteString(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head>`)
	b.WriteString(`<body style="margin:0;padding:0;background-color:` + bgColor + `;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">`)

	// Outer table
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:` + bgColor + `;">`)
	b.WriteString(`<tr><td align="center" style="padding:40px 16px;">`)

	// Card
	b.WriteString(`<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;border-radius:12px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">`)

	// Header
	b.WriteString(`<tr><td style="background-color:` + brandColor + `;padding:28px 40px;text-align:center;">`)
	b.WriteString(`<span style="color:#ffffff;font-size:24px;font-weight:700;letter-spacing:0.5px;">` + html.EscapeString(appName) + `</span>`)
	b.WriteString(`</td></tr>`)

	// Body
	b.WriteString(`<tr><td style="background-color:#ffffff;padding:40px;">`)

	// Greeting
	if p.Greeting != "" {
		b.WriteString(`<p style="margin:0 0 20px;font-size:18px;font-weight:600;color:#18181b;">` + html.EscapeString(p.Greeting) + `</p>`)
	}

	// Body lines
	for _, line := range p.BodyLines {
		b.WriteString(`<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#3f3f46;">` + html.EscapeString(line) + `</p>`)
	}

	// Button
	if p.ButtonText != "" && p.ButtonURL != "" {
		b.WriteString(`<table role="presentation" cellpadding="0" cellspacing="0" style="margin:28px 0;"><tr><td>`)
		b.WriteString(`<a href="` + html.EscapeString(p.ButtonURL) + `" target="_blank" style="display:inline-block;background-color:` + brandColor + `;color:#ffffff;font-size:15px;font-weight:600;text-decoration:none;padding:14px 32px;border-radius:8px;">`)
		b.WriteString(html.EscapeString(p.ButtonText))
		b.WriteString(`</a>`)
		b.WriteString(`</td></tr></table>`)
	}

	// Footer text (inside card)
	if p.FooterText != "" {
		b.WriteString(`<p style="margin:20px 0 0;font-size:13px;line-height:1.5;color:#a1a1aa;">` + html.EscapeString(p.FooterText) + `</p>`)
	}

	b.WriteString(`</td></tr>`)

	// Footer (outside card, inside outer table)
	b.WriteString(`<tr><td style="padding:24px 40px;text-align:center;">`)
	b.WriteString(`<p style="margin:0;font-size:12px;color:#a1a1aa;">` + html.EscapeString(appName) + `</p>`)
	b.WriteString(`</td></tr>`)

	b.WriteString(`</table>`)

	// Close outer table
	b.WriteString(`</td></tr></table>`)
	b.WriteString(`</body></html>`)

	return b.String()
}

func RenderText(p EmailParams) string {
	var b strings.Builder

	if p.Greeting != "" {
		b.WriteString(p.Greeting)
		b.WriteString("\n\n")
	}

	for _, line := range p.BodyLines {
		b.WriteString(line)
		b.WriteString("\n")
	}

	if p.ButtonText != "" && p.ButtonURL != "" {
		b.WriteString("\n")
		b.WriteString(p.ButtonURL)
		b.WriteString("\n")
	}

	if p.FooterText != "" {
		b.WriteString("\n")
		b.WriteString(p.FooterText)
		b.WriteString("\n")
	}

	return b.String()
}
