package perfectshutdown

import "testing"

func TestClose(t *testing.T) {
	ps := New()
	ps.Close()
}
