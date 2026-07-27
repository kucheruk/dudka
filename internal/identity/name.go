package identity

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// DisplayNameFile is the basename of the persisted nick under the data dir.
const DisplayNameFile = "display_name"

// NameResolver picks display_name: flag/prompt → hostname → Прилагательное+Животное.
type NameResolver struct {
	Prompt   func() (string, bool) // ok=false means skipped / unavailable
	Hostname func() (string, bool) // ok=false means missing / unusable
	Generate func() string
}

// Resolve returns the chosen display_name for ROADMAP P011 / DUD-CHAT-110.
func (r NameResolver) Resolve(flagName string) string {
	if n := strings.TrimSpace(flagName); n != "" {
		return n
	}
	if r.Prompt != nil {
		if n, ok := r.Prompt(); ok {
			if n = strings.TrimSpace(n); n != "" {
				return n
			}
		}
	}
	if r.Hostname != nil {
		if n, ok := r.Hostname(); ok && MeaningfulHost(n) {
			return strings.TrimSpace(n)
		}
	}
	if r.Generate != nil {
		return r.Generate()
	}
	return GenerateAdjectiveAnimal(nil)
}

// MeaningfulHost reports whether host can serve as a human-facing nick.
func MeaningfulHost(host string) bool {
	h := strings.TrimSpace(host)
	if h == "" {
		return false
	}
	h = strings.TrimSuffix(h, ".")
	lower := strings.ToLower(h)
	switch lower {
	case "localhost", "localhost.localdomain", "localdomain", "(none)", "none":
		return false
	}
	// Reject pure numeric / empty-looking labels.
	letters := false
	for _, r := range h {
		if unicode.IsLetter(r) {
			letters = true
			break
		}
	}
	return letters
}

var adjectives = []string{
	"Сонный", "Быстрый", "Тихий", "Храбрый", "Уютный",
	"Рыжий", "Северный", "Домашний", "Весёлый", "Мутный",
}

var animals = []string{
	"Барсук", "Ёжик", "Лисица", "Выдра", "Ворон",
	"Кот", "Ёрш", "Суслик", "Хомяк", "Кабан",
}

// GenerateAdjectiveAnimal builds «Прилагательное+Животное».
// pick(n) must return an index in [0,n); nil uses crypto/rand.
func GenerateAdjectiveAnimal(pick func(n int) int) string {
	if pick == nil {
		pick = cryptoPick
	}
	adj := adjectives[pick(len(adjectives))]
	animal := animals[pick(len(animals))]
	return adj + "+" + animal
}

func cryptoPick(n int) int {
	if n <= 0 {
		return 0
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}

// SaveDisplayName writes the nick to dataDir (used by POST /nick / P016).
func SaveDisplayName(dataDir, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("identity: display name is empty")
	}
	if strings.TrimSpace(dataDir) == "" {
		return errors.New("identity: data dir is empty")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("identity: mkdir data dir: %w", err)
	}
	path := filepath.Join(dataDir, DisplayNameFile)
	if err := os.WriteFile(path, []byte(name+"\n"), 0o600); err != nil {
		return fmt.Errorf("identity: write display_name: %w", err)
	}
	return nil
}

// LoadOrCreateDisplayName returns a persisted nick, resolving and saving on first use.
func LoadOrCreateDisplayName(dataDir, flagName string, prompt io.Reader, promptEnabled bool) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", errors.New("identity: data dir is empty")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", fmt.Errorf("identity: mkdir data dir: %w", err)
	}

	path := filepath.Join(dataDir, DisplayNameFile)
	if strings.TrimSpace(flagName) == "" {
		if raw, err := os.ReadFile(path); err == nil {
			if n := strings.TrimSpace(string(raw)); n != "" {
				return n, nil
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("identity: read display_name: %w", err)
		}
	}

	r := NameResolver{
		Hostname: func() (string, bool) {
			h, err := os.Hostname()
			if err != nil {
				return "", false
			}
			return h, true
		},
		Generate: func() string { return GenerateAdjectiveAnimal(nil) },
	}
	if promptEnabled && prompt != nil {
		r.Prompt = func() (string, bool) {
			return readPrompt(prompt)
		}
	}

	name := r.Resolve(flagName)
	if err := os.WriteFile(path, []byte(name+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("identity: write display_name: %w", err)
	}
	return name, nil
}

func readPrompt(r io.Reader) (string, bool) {
	fmt.Fprint(os.Stderr, "Ник (Enter — пропустить): ")
	br := bufio.NewReader(r)
	line, err := br.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", false
	}
	return strings.TrimRight(line, "\r\n"), true
}
