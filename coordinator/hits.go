package coordinator

import (
	"fmt"
	"os"
	"time"

	"btcfind/cluster"
)

const walletsLogFile = "wallets.txt"

func appendWalletHit(path string, keyIndex int, formatID string, formatLabel string, address, wif string, balanceSats uint64) error {
	_, statErr := os.Stat(path)
	newFile := os.IsNotExist(statErr)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if newFile {
		if err := os.Chmod(path, 0600); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(f,
		"--- wallet found %s ---\nkey_index: %d\nformat: %s (%s)\naddress: %s\nbalance_sats: %d\nwif: %s\n\n",
		time.Now().Format(time.RFC3339),
		keyIndex,
		formatID,
		formatLabel,
		address,
		balanceSats,
		wif,
	); err != nil {
		return err
	}
	return f.Sync()
}

func kindFromFormatID(id string) (string, string) {
	switch id {
	case "legacy_compressed":
		return "legacy_compressed", "P2PKH compressed"
	case "legacy_uncompressed":
		return "legacy_uncompressed", "P2PKH uncompressed"
	case "segwit_v0":
		return "segwit_v0", "Native SegWit (bc1q)"
	case "p2sh":
		return "p2sh", "P2SH (3...)"
	case "taproot":
		return "taproot", "Taproot (bc1p)"
	default:
		return id, "Unknown"
	}
}

func (s *Server) handleHit(req cluster.HitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.workers[req.WorkerID]; !ok {
		return fmt.Errorf("unknown worker_id")
	}

	id, label := kindFromFormatID(req.Format)
	if req.Address != "" {
		for _, prev := range s.hitAddresses {
			if prev == req.Address {
				s.ui.Warnf("duplicate hit report for address %s (ignored)\n", req.Address)
				return nil
			}
		}
		s.hitAddresses = append(s.hitAddresses, req.Address)
	}

	if err := appendWalletHit(walletsLogFile, req.KeyIndex, id, label, req.Address, req.WIF, req.BalanceSats); err != nil {
		return err
	}

	s.clusterHits++
	s.ui.PrintClusterHit(req.Address, req.WIF, id, label, req.BalanceSats, req.KeyIndex)
	return nil
}