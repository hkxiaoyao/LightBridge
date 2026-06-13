package binding

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/LightBridge/internal/outbound"
)

type EntityType string

const (
	EntityGlobal   EntityType = "global"
	EntityProvider EntityType = "provider"
	EntityChannel  EntityType = "channel"
	EntityAccount  EntityType = "account"
	EntityRequest  EntityType = "request"
)

type Binding struct {
	ID               int64
	EntityType       EntityType
	EntityID         string
	ProfileID        int64
	Priority         int
	Enabled          bool
	FallbackToDirect bool
}

type Query struct {
	EntityType EntityType
	EntityID   string
}

func QueriesForScope(scope outbound.Scope) []Query {
	queries := make([]Query, 0, 5)
	if requestProfileID(scope) > 0 {
		queries = append(queries, Query{EntityType: EntityRequest, EntityID: requestEntityID(scope)})
	}
	if scope.AccountID > 0 {
		queries = append(queries, Query{EntityType: EntityAccount, EntityID: strconv.FormatInt(scope.AccountID, 10)})
	}
	if scope.ChannelID > 0 {
		queries = append(queries, Query{EntityType: EntityChannel, EntityID: strconv.FormatInt(scope.ChannelID, 10)})
	}
	if providerID := strings.TrimSpace(scope.ProviderID); providerID != "" {
		queries = append(queries, Query{EntityType: EntityProvider, EntityID: providerID})
	}
	queries = append(queries, Query{EntityType: EntityGlobal, EntityID: "default"})
	return queries
}

func requestProfileID(scope outbound.Scope) int64 {
	if scope.Metadata == nil {
		return 0
	}
	value, ok := scope.Metadata["proxy_profile_id"]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		id, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return id
	default:
		return 0
	}
}

func requestEntityID(scope outbound.Scope) string {
	profileID := requestProfileID(scope)
	if profileID <= 0 {
		return ""
	}
	parts := []string{fmt.Sprintf("profile:%d", profileID)}
	if scope.APIKeyID > 0 {
		parts = append(parts, fmt.Sprintf("api_key:%d", scope.APIKeyID))
	}
	if scope.UserID > 0 {
		parts = append(parts, fmt.Sprintf("user:%d", scope.UserID))
	}
	return strings.Join(parts, "|")
}
