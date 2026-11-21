package apipublicwiring

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/constants"
	apipublichttp "github.com/dujiao-next/internal/modules/apipublic/transport/http"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
)

var errAmbiguousGlueConnection = errors.New("ambiguous active dujiao-next site connections")

func NewHandler(c *container.Container) *apipublichttp.Handler {
	var connID uint
	if raw := strings.TrimSpace(os.Getenv("STOREFRONT_ACCOUNT_LINK_CONNECTION_ID")); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, strconv.IntSize); err == nil && parsed > 0 {
			connID = uint(parsed)
		}
	}
	return apipublichttp.NewHandler(glueConnectionResolver{
		c:                      c,
		configuredConnectionID: connID,
	})
}

type glueConnectionResolver struct {
	c                      *container.Container
	configuredConnectionID uint
}

func (r glueConnectionResolver) ResolveGlueBaseURL(_ context.Context) (string, bool, error) {
	if r.c == nil {
		return "", false, nil
	}
	var conn *siteconnectiondomain.Connection
	if r.configuredConnectionID > 0 {
		if r.c.SiteConnectionService == nil {
			return "", false, nil
		}
		found, err := r.c.SiteConnectionService.GetByID(r.configuredConnectionID)
		if err != nil {
			return "", false, err
		}
		if found == nil || found.Status != constants.ConnectionStatusActive || found.Protocol != constants.ConnectionProtocolDujiaoNext {
			return "", false, nil
		}
		conn = found
	} else {
		if r.c.SiteConnectionRepo == nil {
			return "", false, nil
		}
		active, err := r.c.SiteConnectionRepo.ListActive()
		if err != nil {
			return "", false, err
		}
		var matches []siteconnectiondomain.Connection
		for _, item := range active {
			if item.Protocol == constants.ConnectionProtocolDujiaoNext {
				matches = append(matches, item)
			}
		}
		if len(matches) == 0 {
			return "", false, nil
		}
		if len(matches) > 1 {
			return "", false, errAmbiguousGlueConnection
		}
		conn = &matches[0]
	}

	baseURL := strings.TrimRight(strings.TrimSpace(conn.BaseURL), "/")
	if baseURL == "" {
		return "", false, nil
	}
	return baseURL, true, nil
}
