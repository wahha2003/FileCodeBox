package httpurl

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
)

const (
	defaultAPIBaseURL    = "http://api.localhost:12345"
	defaultPublicBaseURL = "http://localhost:3000"
)

func trimTrailingSlash(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func requestBaseURL(c *app.RequestContext) string {
	if c == nil {
		return ""
	}

	scheme := string(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		scheme = string(c.Request.Scheme())
	}
	if scheme == "" {
		scheme = "http"
	}

	host := string(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = string(c.Host())
	}
	if host == "" {
		return ""
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}

func APIBaseURL(c *app.RequestContext) string {
	if cfg := conf.GetGlobalConfig(); cfg != nil && cfg.Server.BaseURL != "" {
		return trimTrailingSlash(cfg.Server.BaseURL)
	}

	if requestBase := requestBaseURL(c); requestBase != "" {
		return trimTrailingSlash(requestBase)
	}

	return defaultAPIBaseURL
}

func PublicBaseURL(c *app.RequestContext) string {
	if cfg := conf.GetGlobalConfig(); cfg != nil && cfg.Server.PublicBaseURL != "" {
		return trimTrailingSlash(cfg.Server.PublicBaseURL)
	}

	if c != nil {
		if origin := trimTrailingSlash(string(c.GetHeader("Origin"))); origin != "" {
			return origin
		}

		if referer := string(c.GetHeader("Referer")); referer != "" {
			if parsed, err := url.Parse(referer); err == nil && parsed.Scheme != "" && parsed.Host != "" {
				return trimTrailingSlash(fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host))
			}
		}
	}

	apiBase := APIBaseURL(c)
	switch {
	case strings.Contains(apiBase, "://api.localhost"):
		return defaultPublicBaseURL
	case strings.Contains(apiBase, "://api."):
		return trimTrailingSlash(strings.Replace(apiBase, "://api.", "://", 1))
	default:
		return defaultPublicBaseURL
	}
}

func BuildAPIURL(c *app.RequestContext, path string) string {
	return fmt.Sprintf("%s/%s", APIBaseURL(c), strings.TrimLeft(path, "/"))
}

func BuildAPIDownloadURL(c *app.RequestContext, code string, password string) string {
	query := url.Values{}
	query.Set("code", code)
	if password != "" {
		query.Set("password", password)
	}

	return fmt.Sprintf("%s/share/download?%s", APIBaseURL(c), query.Encode())
}

func BuildPublicShareURL(c *app.RequestContext, code string) string {
	return fmt.Sprintf("%s/#/share/%s", PublicBaseURL(c), url.PathEscape(code))
}
