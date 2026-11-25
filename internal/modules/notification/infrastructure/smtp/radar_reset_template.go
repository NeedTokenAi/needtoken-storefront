package smtp

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/dujiao-next/internal/i18n"
	"github.com/dujiao-next/internal/shared/mailbrand"
	"github.com/dujiao-next/internal/telegramidentity"
)

var radarResetEmailTemplate = template.Must(template.New("radar_reset").Parse(`<!doctype html>
<html lang="{{.Locale}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}}</title></head>
<body style="margin:0;padding:0;background:#f5f7f8;color:#172321;font-family:Arial,'Microsoft YaHei',sans-serif">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f5f7f8"><tr><td align="center" style="padding:36px 16px">
<table role="presentation" width="540" cellpadding="0" cellspacing="0" style="width:100%;max-width:540px;background:#ffffff;border:1px solid #e3eae7;border-radius:16px"><tr><td style="padding:32px">
<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="width:34px;height:34px;text-align:center;border-radius:9px;background:#dff8ef;color:#087e60;font-size:21px;font-weight:bold">{{.Mark}}</td><td style="padding-left:10px;font-size:18px;font-weight:bold">{{.Brand}}</td></tr></table>
<p style="margin:24px 0 8px;font-size:12px;font-weight:700;letter-spacing:1px;text-transform:uppercase;color:#087e60">{{.Badge}}</p>
<h1 style="margin:0 0 12px;font-size:22px;line-height:1.4;font-weight:700">{{.Title}}</h1>
<p style="margin:0 0 20px;font-size:14px;line-height:1.75;color:#50605a">{{.Intro}}</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4faf7;border:1px solid #d7eee4;border-radius:12px"><tr><td style="padding:16px 18px">
<p style="margin:0 0 8px;font-size:12px;color:#50605a"><strong>{{.TimeLabel}}</strong> {{.AnnouncedAt}}{{if .Scope}} · {{.Scope}}{{end}}</p>
<p style="margin:0;font-size:14px;line-height:1.7;color:#172321;white-space:pre-line">{{.Summary}}</p>
</td></tr></table>
<table role="presentation" cellpadding="0" cellspacing="0" style="margin-top:22px"><tr>
{{if .RadarURL}}<td style="padding-right:10px"><a href="{{.RadarURL}}" style="display:inline-block;padding:10px 18px;border-radius:9px;background:#087e60;color:#ffffff;font-size:13px;font-weight:600;text-decoration:none">{{.OpenRadarLabel}}</a></td>{{end}}
{{if .SourceURL}}<td><a href="{{.SourceURL}}" style="display:inline-block;padding:10px 16px;border-radius:9px;border:1px solid #d2ddd8;color:#172321;font-size:13px;font-weight:600;text-decoration:none">{{.SourceLabel}}</a></td>{{end}}
</tr></table>
<p style="margin:24px 0 0;padding-top:18px;border-top:1px solid #e8eeeb;font-size:12px;line-height:1.7;color:#697770">{{.Footer}}</p>
</td></tr></table>
</td></tr></table></body></html>`))

// SendRadarResetEmail notifies a subscribed member that a tracked provider has issued a quota reset.
func (s *Service) SendRadarResetEmail(toEmail, locale, channel, recordType, scope, summary, sourceURL string, announcedAt time.Time, brand mailbrand.Brand) error {
	if telegramidentity.IsPlaceholderEmail(toEmail) {
		return nil
	}
	normalized := normalizeLocale(locale)
	brand.SiteURL = verificationSiteURL(brand.SiteURL)
	siteName := strings.TrimSpace(brand.SiteName)
	if siteName == "" {
		siteName = "NeedToken"
	}
	mark := "N"
	if runes := []rune(siteName); len(runes) > 0 {
		mark = strings.ToUpper(string(runes[0]))
	}

	channelLabel := radarChannelDisplayName(channel)
	typeLabel := radarRecordTypeLabel(normalized, channel, recordType)
	announcedText := announcedAt.UTC().Format("2006-01-02 15:04 UTC")
	radarURL := ""
	if brand.SiteURL != "" {
		radarURL = brand.SiteURL + "/tools/radar?channel=" + channel
	}
	safeSourceURL := verificationSiteURL(sourceURL)
	if summary == "" {
		summary = typeLabel
	}

	var subject, title, intro, timeLabel, openRadarLabel, sourceLabel, footer string
	switch normalized {
	case i18n.LocaleEN:
		subject = fmt.Sprintf("[%s] %s Quota Reset Detected (%s)", siteName, channelLabel, typeLabel)
		title = fmt.Sprintf("%s Subscription Reset Notification", channelLabel)
		intro = fmt.Sprintf("A new %s reset announcement has been detected on the Reset Radar.", channelLabel)
		timeLabel = "Announced:"
		openRadarLabel = "Open Reset Radar"
		sourceLabel = "View Source"
		footer = "You received this email because you subscribed to Reset Radar alerts. You can manage or unsubscribe anytime on the Reset Radar page."
	case i18n.LocaleTW:
		subject = fmt.Sprintf("[%s] %s 訂閱已重置通知（%s）", siteName, channelLabel, typeLabel)
		title = fmt.Sprintf("%s 訂閱已重置", channelLabel)
		intro = fmt.Sprintf("重置雷達偵測到 %s 發布了最新的額度重置公告，以下為通知詳情：", channelLabel)
		timeLabel = "公告時間："
		openRadarLabel = "前往重置雷達"
		sourceLabel = "查看來源"
		footer = "您收到這封郵件是因為已訂閱重置雷達通知。您隨時可在重置雷達頁面調整或取消訂閱。"
	default:
		subject = fmt.Sprintf("[%s] %s 订阅已重置通知（%s）", siteName, channelLabel, typeLabel)
		title = fmt.Sprintf("%s 订阅已重置", channelLabel)
		intro = fmt.Sprintf("重置雷达检测到 %s 发布了最新的额度重置公告，以下为通知详情：", channelLabel)
		timeLabel = "公告时间："
		openRadarLabel = "前往重置雷达"
		sourceLabel = "查看来源"
		footer = "您收到这封邮件是因为已订阅重置雷达消息。您随时可在重置雷达页面管理或取消订阅。"
	}

	plainParts := []string{
		title,
		"",
		intro,
		fmt.Sprintf("%s %s", timeLabel, announcedText),
	}
	if scope != "" {
		plainParts = append(plainParts, scope)
	}
	plainParts = append(plainParts, "", summary)
	if radarURL != "" {
		plainParts = append(plainParts, "", openRadarLabel+": "+radarURL)
	}
	if safeSourceURL != "" {
		plainParts = append(plainParts, sourceLabel+": "+safeSourceURL)
	}
	plainParts = append(plainParts, "", footer)
	plainBody := strings.Join(plainParts, "\n")

	var htmlBuf bytes.Buffer
	err := radarResetEmailTemplate.Execute(&htmlBuf, struct {
		Locale         string
		Brand          string
		Mark           string
		Badge          string
		Title          string
		Intro          string
		TimeLabel      string
		AnnouncedAt    string
		Scope          string
		Summary        string
		RadarURL       string
		SourceURL      string
		OpenRadarLabel string
		SourceLabel    string
		Footer         string
	}{
		Locale:         normalized,
		Brand:          siteName,
		Mark:           mark,
		Badge:          fmt.Sprintf("%s · %s", channelLabel, typeLabel),
		Title:          title,
		Intro:          intro,
		TimeLabel:      timeLabel,
		AnnouncedAt:    announcedText,
		Scope:          scope,
		Summary:        summary,
		RadarURL:       radarURL,
		SourceURL:      safeSourceURL,
		OpenRadarLabel: openRadarLabel,
		SourceLabel:    sourceLabel,
		Footer:         footer,
	})
	if err != nil {
		return err
	}

	from, addr, err := s.prepareSMTPEnvelope(toEmail, brand.FromName)
	if err != nil {
		return err
	}
	message, err := buildAlternativeEmailMessage(from, toEmail, subject, plainBody, htmlBuf.String(), brand.ReplyTo)
	if err != nil {
		return err
	}
	return s.sendSMTPMessage(addr, toEmail, []byte(message))
}

func radarChannelDisplayName(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "codex":
		return "Codex"
	case "claude":
		return "Claude"
	case "gemini":
		return "Gemini"
	case "grok":
		return "Grok"
	default:
		return channel
	}
}

func radarRecordTypeLabel(locale, channel, recordType string) string {
	isBanked := strings.EqualFold(strings.TrimSpace(recordType), "banked")
	isGemini := strings.EqualFold(strings.TrimSpace(channel), "gemini")
	switch locale {
	case i18n.LocaleEN:
		if isBanked {
			if isGemini {
				return "Model Refresh"
			}
			return "Banked Reset"
		}
		return "Full Reset"
	case i18n.LocaleTW:
		if isBanked {
			if isGemini {
				return "版本更新"
			}
			return "重置卡"
		}
		return "全員重置"
	default:
		if isBanked {
			if isGemini {
				return "版本更新"
			}
			return "重置卡"
		}
		return "全员重置"
	}
}
