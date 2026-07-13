package httpapi

import (
	"net/http/cookiejar"
	"testing"
)

// newCookieJar builds a cookie jar or fails the test.
func newCookieJar(t *testing.T) *cookiejar.Jar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	return jar
}
