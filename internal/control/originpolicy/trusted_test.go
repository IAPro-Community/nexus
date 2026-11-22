package originpolicy

import "testing"

// A tailnet address (Tailscale allocates inside 100.64.0.0/10) is private and
// must be accepted as a browser origin target, exactly like an RFC1918 address.
// The bind policy already allows that range; the Origin policy used to reject
// it, which made the private-overlay path unusable.
func TestValidateAcceptsPrivateOverlayAddresses(t *testing.T) {
	cases := []struct {
		host, origin string
		want         bool
	}{
		{"100.101.102.103:13000", "https://100.101.102.103:13000", true},
		{"100.64.0.1:13000", "https://100.64.0.1:13000", true},
		{"100.127.255.254:13000", "https://100.127.255.254:13000", true},
		{"10.0.0.5:13000", "https://10.0.0.5:13000", true},
		{"172.16.0.5:13000", "https://172.16.0.5:13000", true},
		{"[fd00::1]:13000", "https://[fd00::1]:13000", true},
		// The CGNAT range ends at 100.127.255.255.
		{"100.128.0.1:13000", "https://100.128.0.1:13000", false},
		{"8.8.8.8:13000", "https://8.8.8.8:13000", false},
	}
	for _, tc := range cases {
		if got := Validate(tc.host, tc.origin); got != tc.want {
			t.Fatalf("host=%q origin=%q got=%v want=%v", tc.host, tc.origin, got, tc.want)
		}
	}
}

// The exact-host rule still applies to private addresses: a private target
// does not make a mismatched Origin acceptable.
func TestValidateKeepsExactHostRuleForPrivateAddresses(t *testing.T) {
	if Validate("100.101.102.103:13000", "https://100.101.102.104:13000") {
		t.Fatal("a mismatched origin host must stay rejected on a private address")
	}
	if Validate("10.0.0.5:13000", "https://10.0.0.6:13000") {
		t.Fatal("a mismatched origin host must stay rejected on a private address")
	}
}

// A private, stable hostname (tailnet name or private DNS entry) is accepted
// only when the operator publishes it as the remote origin.
//
// The publication defines one exact address: same scheme, same hostname, same
// effective port. A different port on the same hostname is a different
// service, and a different scheme is a different trust decision.
func TestValidateAcceptsPublishedRemoteOrigin(t *testing.T) {
	t.Setenv(remoteOriginEnv, "")
	if Validate("nexus.tail1234.ts.net:13000", "https://nexus.tail1234.ts.net:13000") {
		t.Fatal("an unpublished hostname must stay rejected")
	}

	t.Setenv(remoteOriginEnv, "https://nexus.tail1234.ts.net:13000")
	if !Validate("nexus.tail1234.ts.net:13000", "https://nexus.tail1234.ts.net:13000") {
		t.Fatal("a published hostname must be accepted")
	}
	// The publication widens which host is acceptable, never who may send it.
	if Validate("nexus.tail1234.ts.net:13000", "https://other.tail1234.ts.net:13000") {
		t.Fatal("a published hostname must not trust a different host")
	}
}

// The published port is part of the published address: another port on the
// same hostname is another service and must not inherit the trust.
func TestPublishedRemoteOriginDoesNotTrustAnotherPort(t *testing.T) {
	t.Setenv(remoteOriginEnv, "https://nexus.tail1234.ts.net:8443")
	if Validate("nexus.tail1234.ts.net:9443", "https://nexus.tail1234.ts.net:9443") {
		t.Fatal("a published origin must not trust another port")
	}
	if !Validate("nexus.tail1234.ts.net:8443", "https://nexus.tail1234.ts.net:8443") {
		t.Fatal("the published port must be accepted")
	}
}

// The scheme is part of the published address too: the same hostname served
// over plain HTTP is a different trust decision.
func TestPublishedRemoteOriginDoesNotTrustAnotherScheme(t *testing.T) {
	t.Setenv(remoteOriginEnv, "https://nexus.tail1234.ts.net")
	if Validate("nexus.tail1234.ts.net", "http://nexus.tail1234.ts.net") {
		t.Fatal("a published origin must not trust another scheme")
	}
}

// A trusted proxy that terminates TLS may rewrite the request Host to the
// local service, so the browser origin and the request Host stop being the
// same string. The publication is what restores the binding: the browser
// origin must be exactly the published address.
func TestPublishedRemoteOriginAllowsAProxyRewrittenHost(t *testing.T) {
	t.Setenv(remoteOriginEnv, "https://nexus.tail1234.ts.net")
	if !Validate("127.0.0.1:13000", "https://nexus.tail1234.ts.net") {
		t.Fatal("a published origin must survive a proxy that rewrites the request host")
	}
	// The rewrite must not open the door to a different browser origin.
	if Validate("127.0.0.1:13000", "https://other.tail1234.ts.net") {
		t.Fatal("a rewritten host must not accept an unpublished browser origin")
	}
	// Nor may it cover a request that landed on a public interface.
	if Validate("8.8.8.8:13000", "https://nexus.tail1234.ts.net") {
		t.Fatal("a published private origin must not cover a public request host")
	}
}

// A malformed publication must not widen the policy: a wildcard, a path, a
// scheme-less value or a non-HTTP scheme is ignored.
func TestRemoteOriginRejectsMalformedValues(t *testing.T) {
	invalid := []string{
		"nexus.tail1234.ts.net",
		"ftp://nexus.tail1234.ts.net",
		"https://*.tail1234.ts.net",
		"https://nexus.tail1234.ts.net/path",
		"https://nexus.tail1234.ts.net?x=1",
		"https://user@nexus.tail1234.ts.net",
		"https://",
	}
	for _, value := range invalid {
		t.Setenv(remoteOriginEnv, value)
		if _, ok := RemoteOrigin(); ok {
			t.Fatalf("value %q must not be accepted as a remote origin", value)
		}
		if Validate("nexus.tail1234.ts.net:13000", "https://nexus.tail1234.ts.net:13000") {
			t.Fatalf("value %q must not widen the origin policy", value)
		}
	}
}

// The published origin is the address the operator shares when pairing a
// device, so it must round-trip in a normalized form.
func TestRemoteOriginIsNormalized(t *testing.T) {
	t.Setenv(remoteOriginEnv, "  HTTPS://Nexus.Tail1234.ts.NET/  ")
	origin, ok := RemoteOrigin()
	if !ok {
		t.Fatal("a valid origin must be accepted")
	}
	if origin != "https://nexus.tail1234.ts.net" {
		t.Fatalf("origin = %q", origin)
	}

	t.Setenv(remoteOriginEnv, "https://nexus.tail1234.ts.net:8443")
	origin, ok = RemoteOrigin()
	if !ok {
		t.Fatal("an origin with a port must be accepted")
	}
	if origin != "https://nexus.tail1234.ts.net:8443" {
		t.Fatalf("origin = %q", origin)
	}
}
