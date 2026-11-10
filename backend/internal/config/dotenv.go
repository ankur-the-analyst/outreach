package config

import (
	"bufio"
	"os"
	"strings"
)

func loadDotenv() {
	for _, path := range []string{".env", "../.env"} {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, val, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			if key == "" {
				continue
			}
			if _, exists := os.LookupEnv(key); exists {
				continue
			}
			_ = os.Setenv(key, val)
		}
		_ = f.Close()
	}
}
