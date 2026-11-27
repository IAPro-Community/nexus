package spec

import sdkprovider "github.com/IAPro-Community/IAPro-SDK/providers/spec"

func init() {
	// Product availability policy only: keep the SDK Gemini spec intact in
	// other products while Nexus excludes it from its own process registry.
	sdkprovider.Unregister("gemini")
}
