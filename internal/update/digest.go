package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// VerifyFileDigest checks path against a GitHub-style digest ("sha256:<hex>").
// An empty digest is a no-op (older releases / mock servers).
func VerifyFileDigest(path, digest string) error {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return nil
	}
	i := strings.IndexByte(digest, ':')
	if i < 0 {
		return fmt.Errorf("unsupported digest format: %s", digest)
	}
	algo := strings.ToLower(digest[:i])
	wantHex := strings.ToLower(digest[i+1:])
	if algo != "sha256" {
		return fmt.Errorf("unsupported digest algorithm: %s", algo)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != wantHex {
		return fmt.Errorf("sha256 mismatch: got %s want %s", got, wantHex)
	}
	return nil
}
