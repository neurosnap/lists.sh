package internal

import (
	"fmt"
	"html/template"
	"log"

	"github.com/picosh/cms/config"
	"go.uber.org/zap"
)

type SitePageData struct {
	Domain  template.URL
	HomeURL template.URL
	Email   string
}

type ConfigSite struct {
	*config.ConfigCms
}

func NewConfigSite() *ConfigSite {
	domain := GetEnv("LISTS_DOMAIN", "lists.sh")
	email := GetEnv("LISTS_EMAIL", "support@lists.sh")
	subdomains := GetEnv("LISTS_SUBDOMAINS", "0")
	dbURL := GetEnv("DATABASE_URL", "")
	subdomainsEnabled := false
	if subdomains == "1" {
		subdomainsEnabled = true
	}

	return &ConfigSite{
		&config.ConfigCms{
			Domain:            domain,
			Email:             email,
			SubdomainsEnabled: subdomainsEnabled,
			DbURL:             dbURL,
		},
	}
}

func (c *ConfigSite) GetSiteData() *SitePageData {
	return &SitePageData{
		Domain:  template.URL(c.Domain),
		HomeURL: template.URL(c.HomeURL()),
		Email:   c.Email,
	}
}

func (c *ConfigSite) RssBlogURL(username string) string {
	if c.IsSubdomains() {
		return fmt.Sprintf("//%s.%s/rss", username, c.Domain)
	}

	return fmt.Sprintf("/%s/rss", username)
}

func (c *ConfigSite) HomeURL() string {
	if c.IsSubdomains() {
		return fmt.Sprintf("//%s", c.Domain)
	}

	return "/"
}

func (c *ConfigSite) ReadURL() string {
	if c.IsSubdomains() {
		return fmt.Sprintf("https://%s/read", c.Domain)
	}

	return "/read"
}

func (c *ConfigSite) CreateLogger() *zap.SugaredLogger {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	return logger.Sugar()
}
