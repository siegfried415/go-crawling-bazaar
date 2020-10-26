package node

import "github.com/siegfried415/gdf-rebuild/wallet"

// WalletSubmodule enhances the `Node` with a "Wallet" and FIL transfer capabilities.
type WalletSubmodule struct {
	Wallet *wallet.Wallet
}
