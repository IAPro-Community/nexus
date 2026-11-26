package spec

import (
	sdkprovider "github.com/IAPro-Community/IAPro-SDK/providers/spec"
)

func init() {
	// Extend Cursor with chat history directory sync while keeping all
	// credential files strictly isolated per profile.
	_ = sdkprovider.Extend("cursor", func(s *sdkprovider.Spec) {
		s.HistorySyncPolicy = sdkprovider.HistorySyncPolicy{
			Mode:            sdkprovider.HistorySyncLink,
			RootSubdir:      ".cursor",
			SharedPaths:     []string{"chats"},
			CredentialPaths: []string{"auth.json", "cli-config.json"},
		}
	})
}
