package alerting

import (
	"errors"
	"strings"
)

const tenantRefSeparator = "."

func ComposeTenantRef(clientId string, value string) string {
	return clientId + tenantRefSeparator + value
}

func SplitTenantRef(ref string) (string, string, error) {
	parts := strings.SplitN(ref, tenantRefSeparator, 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("invalid tenant reference")
	}
	return parts[0], parts[1], nil
}
