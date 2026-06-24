package model

// UserInfoPayload holds the OIDC UserInfo claims relevant to the BFF.
// The JSON field names match the exact claim keys returned by Keycloak/Zitadel.
type UserInfoPayload struct {
	Sub            string   `json:"sub"`
	Email          string   `json:"email"`
	TenantID       string   `json:"tenantid"`
	SubscriptionID string   `json:"subscriptionid"`
	Name           string   `json:"name"`
	Roles          []string `json:"roles"`
}

// ConfiguratorViewModel is a placeholder aggregate view-model for the configurator
// UI namespace (/api/bff/v1/configurator/*). Fields will be extended by subsequent
// feature specs as specific configurator capabilities are implemented.
type ConfiguratorViewModel struct{}

// AuditorViewModel is a placeholder aggregate view-model for the auditor
// UI namespace (/api/bff/v1/auditor/*). Fields will be extended by subsequent
// feature specs as specific auditor capabilities are implemented.
type AuditorViewModel struct{}
