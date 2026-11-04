package analyzer

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"sort"
	"strings"
)

var hashFactories = map[string]func() hash.Hash{
	"md5":    md5.New,
	"sha1":   sha1.New,
	"sha256": sha256.New,
	"sha512": sha512.New,
}

// HashAlgorithms returns the supported algorithm names, sorted.
func HashAlgorithms() []string {
	names := make([]string, 0, len(hashFactories))
	for n := range hashFactories {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ParseAlgorithms splits a comma-separated list of algorithm names,
// normalises case, removes duplicates and validates each name.
func ParseAlgorithms(list string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, a := range strings.Split(list, ",") {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" || seen[a] {
			continue
		}
		if _, ok := hashFactories[a]; !ok {
			return nil, fmt.Errorf("unknown hash algorithm %q (supported: %s)",
				a, strings.Join(HashAlgorithms(), ", "))
		}
		seen[a] = true
		out = append(out, a)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no hash algorithm given")
	}
	return out, nil
}

// HashResult holds the digests computed for a file.
type HashResult struct {
	Path    string            `json:"path"`
	Digests map[string]string `json:"digests"`
}

// HashFile computes the requested digests of the file at path in one pass.
// Algorithm names are validated with ParseAlgorithms semantics.
func HashFile(path string, algos []string) (HashResult, error) {
	algos, err := ParseAlgorithms(strings.Join(algos, ","))
	if err != nil {
		return HashResult{}, err
	}
	hashers := make(map[string]hash.Hash, len(algos))
	writers := make([]io.Writer, 0, len(algos))
	for _, a := range algos {
		h := hashFactories[a]()
		hashers[a] = h
		writers = append(writers, h)
	}

	f, err := os.Open(path)
	if err != nil {
		return HashResult{}, err
	}
	defer f.Close()

	if _, err := io.Copy(io.MultiWriter(writers...), f); err != nil {
		return HashResult{}, err
	}

	res := HashResult{Path: path, Digests: make(map[string]string, len(hashers))}
	for name, h := range hashers {
		res.Digests[name] = hex.EncodeToString(h.Sum(nil))
	}
	return res, nil
}
