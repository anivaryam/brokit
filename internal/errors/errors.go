package errors

import (
	"errors"
	"fmt"
	"net"
)

func WrapNetworkError(err error) error {
	if err == nil {
		return nil
	}
	var dnsErr *net.DNSError
	var opErr *net.OpError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("network error: cannot reach %s — check your internet connection", dnsErr.Name)
	}
	if errors.As(err, &opErr) {
		return fmt.Errorf("network error: %s — check your internet connection", opErr.Op)
	}
	return err
}