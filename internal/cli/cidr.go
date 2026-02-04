package cli

import (
	"eosrift.com/eosrift/internal/control"
)

func validateCIDRs(field string, values []string) error {
	_, err := control.ParseCIDRList(field, values, 0)
	return err
}
