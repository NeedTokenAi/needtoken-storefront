package smtp

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/i18n"
	"github.com/dujiao-next/internal/shared/mailbrand"
)

// A native, asset-free transactional layout inspired by React Email's Barebone
// activation email. No React runtime, remote images, fonts or tracking are sent.
var verificationEmail = template.Must(template.New("verification").Parse(`<!doctype html>
<html lang="{{.Locale}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}}</title></head>
<body style="margin:0;padding:0;background:#f5f7f8;color:#172321;font-family:Arial,'Microsoft YaHei',sans-serif">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f5f7f8"><tr><td align="center" style="padding:40px 16px">
<table role="presentation" width="520" cellpadding="0" cellspacing="0" style="width:100%;max-width:520px;background:#ffffff;border:1px solid #e3eae7;border-radius:16px"><tr><td style="padding:32px">
<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="width:34px;height:34px;text-align:center;border-radius:9px;background:#dff8ef;color:#087e60;font-size:23px;font-weight:bold">{{.Mark}}</td><td style="padding-left:10px;font-size:19px;font-weight:bold">{{.Brand}}</td></tr></table>
<h1 style="margin:32px 0 14px;font-size:26px;line-height:1.4;font-weight:700">{{.Title}}</h1>
<p style="margin:0 0 24px;font-size:15px;line-height:1.8;color:#50605a">{{.Intro}}</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:22px 8px;background:#effaf6;border:1px solid #d7eee4;border-radius:10px;font-size:32px;line-height:1.4;font-weight:bold;letter-spacing:7px;color:#087e60;font-family:Consolas,monospace">{{.Code}}</td></tr></table>
<p style="margin:16px 0 24px;font-size:13px;line-height:1.7;color:#50605a">{{.Expiry}}</p>
<p style="margin:0;padding-top:22px;border-top:1px solid #e8eeeb;font-size:13px;line-height:1.8;color:#697770">{{.Safety}}</p>
{{if .URL}}<p style="margin:18px 0 0;font-size:12px;line-height:1.6"><a href="{{.URL}}" style="color:#087e60;text-decoration:underline">{{.URL}}</a></p>{{end}}
</td></tr></table><p style="margin:18px 0 0;font-size:12px;line-height:1.7;color:#79857f">{{.Footer}}</p>
</td></tr></table></body></html>`))

func verificationSiteURL(raw string) string {
	value, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || value.Host == "" || value.User != nil || (value.Scheme != "https" && value.Scheme != "http") {
		return ""
	}
	return strings.TrimRight(value.String(), "/")
}

func verificationExpiry(locale string, minutes int) string {
	switch normalizeLocale(locale) {
	case i18n.LocaleEN:
		return fmt.Sprintf("This code expires in %d minutes. Do not share it with anyone.", minutes)
	case i18n.LocaleTW:
		return fmt.Sprintf("驗證碼 %d 分鐘內有效，請勿向任何人透露。", minutes)
	default:
		return fmt.Sprintf("验证码 %d 分钟内有效，请勿向任何人透露。", minutes)
	}
}

func verificationExpiryMinutes(cfg *config.EmailConfig, purpose string) int {
	minutes := 10
	if cfg != nil && cfg.VerifyCode.ExpireMinutes > 0 {
		minutes = cfg.VerifyCode.ExpireMinutes
	}
	if strings.EqualFold(strings.TrimSpace(purpose), constants.VerifyPurposeEmailAuth) {
		return min(minutes, constants.EmailAuthMaxExpireMinutes)
	}
	return minutes
}

func renderVerificationEmail(code, purpose, locale string, brand mailbrand.Brand, expiry int) (string, error) {
	normalized := normalizeLocale(locale)
	title, _ := buildVerifyCodeContent("", purpose, normalized)
	data := struct{ Locale, Title, Intro, Safety, Footer, Brand, Mark, Code, Expiry, URL string }{
		Locale: normalized, Title: title, Brand: strings.TrimSpace(brand.SiteName), Code: code,
		Expiry: verificationExpiry(normalized, expiry), URL: verificationSiteURL(brand.SiteURL),
	}
	switch normalized {
	case i18n.LocaleEN:
		data.Intro = "Enter the code on the page where you requested it to continue."
		data.Safety = "If you did not request this email, you can ignore it. Keep this code private."
		data.Footer = "Account security email · Please do not reply with your code."
		if data.Brand == "" {
			data.Brand = "Account service"
		}
	case i18n.LocaleTW:
		data.Intro = "請回到剛才的頁面，輸入以下驗證碼繼續操作。"
		data.Safety = "若不是您本人發起，請忽略這封郵件。請勿轉發或透露驗證碼。"
		data.Footer = "帳戶安全郵件 · 請勿回覆或傳送驗證碼"
		if data.Brand == "" {
			data.Brand = "帳戶服務"
		}
	default:
		data.Intro = "请回到刚才的页面，输入以下验证码继续操作。"
		data.Safety = "若不是您本人发起，请忽略这封邮件。请勿转发或透露验证码。"
		data.Footer = "账户安全邮件 · 请勿回复或发送验证码"
		if data.Brand == "" {
			data.Brand = "账户服务"
		}
	}
	data.Mark = string([]rune(data.Brand)[0])
	var rendered bytes.Buffer
	if err := verificationEmail.Execute(&rendered, data); err != nil {
		return "", err
	}
	return rendered.String(), nil
}

func buildAlternativeEmailMessage(from, to, subject, textBody, htmlBody, replyTo string) (string, error) {
	var message bytes.Buffer
	writer := multipart.NewWriter(&message)
	writeStandardHeaders(&message, from, to, subject, replyTo)
	fmt.Fprintf(&message, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", writer.Boundary())
	for _, part := range []struct{ mediaType, body string }{{"text/plain", textBody}, {"text/html", htmlBody}} {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Type", part.mediaType+"; charset=UTF-8")
		header.Set("Content-Transfer-Encoding", "quoted-printable")
		body, err := writer.CreatePart(header)
		if err != nil {
			return "", err
		}
		encoded := quotedprintable.NewWriter(body)
		if _, err := io.WriteString(encoded, part.body); err != nil {
			return "", err
		}
		if err := encoded.Close(); err != nil {
			return "", err
		}
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	return message.String(), nil
}
