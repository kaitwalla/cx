package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// AddHost adds a new host to ~/.ssh/config
func AddHost(host Host) error {
	// Check if host already exists
	existing, err := FindHost(host.Alias)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("host %q already exists", host.Alias)
	}

	path := ConfigPath()

	// Open file for appending
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add a newline before the host block if file is not empty
	info, _ := f.Stat()
	if info.Size() > 0 {
		f.WriteString("\n")
	}

	// Write host block
	f.WriteString(fmt.Sprintf("Host %s\n", host.Alias))
	if host.HostName != "" {
		f.WriteString(fmt.Sprintf("    HostName %s\n", host.HostName))
	}
	if host.User != "" {
		f.WriteString(fmt.Sprintf("    User %s\n", host.User))
	}
	if host.Port != "" && host.Port != "22" {
		f.WriteString(fmt.Sprintf("    Port %s\n", host.Port))
	}
	if host.IdentityFile != "" {
		f.WriteString(fmt.Sprintf("    IdentityFile %s\n", host.IdentityFile))
	}

	return nil
}

// UpdateHost updates an existing host in ~/.ssh/config
func UpdateHost(oldAlias string, host Host) error {
	path := ConfigPath()

	// Read entire file
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var result []string
	inTargetHost := false
	skipUntilNextHost := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is a Host line
		if strings.HasPrefix(strings.ToLower(trimmed), "host ") {
			parts := strings.SplitN(trimmed, " ", 2)
			if len(parts) == 2 {
				alias := strings.TrimSpace(parts[1])
				if alias == oldAlias {
					inTargetHost = true
					skipUntilNextHost = true
					// Write the new host block
					result = append(result, fmt.Sprintf("Host %s", host.Alias))
					if host.HostName != "" {
						result = append(result, fmt.Sprintf("    HostName %s", host.HostName))
					}
					if host.User != "" {
						result = append(result, fmt.Sprintf("    User %s", host.User))
					}
					if host.Port != "" && host.Port != "22" {
						result = append(result, fmt.Sprintf("    Port %s", host.Port))
					}
					if host.IdentityFile != "" {
						result = append(result, fmt.Sprintf("    IdentityFile %s", host.IdentityFile))
					}
					continue
				} else {
					inTargetHost = false
					skipUntilNextHost = false
				}
			}
		}

		// Skip lines belonging to the old host
		if skipUntilNextHost {
			if strings.HasPrefix(strings.ToLower(trimmed), "host ") {
				skipUntilNextHost = false
				inTargetHost = false
			} else if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				continue
			}
		}

		if !skipUntilNextHost || inTargetHost {
			if !skipUntilNextHost {
				result = append(result, line)
			}
		}
	}

	// Write back
	return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0600)
}

// DeleteHost removes a host from ~/.ssh/config
func DeleteHost(alias string) error {
	path := ConfigPath()

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	var result []string
	inTargetHost := false
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check if this is a Host line
		if strings.HasPrefix(strings.ToLower(trimmed), "host ") {
			parts := strings.SplitN(trimmed, " ", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[1]) == alias {
				inTargetHost = true
				continue
			} else {
				inTargetHost = false
			}
		}

		// Skip lines belonging to the target host
		if inTargetHost {
			// Check if this line is a config directive (starts with whitespace and has content)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
				continue
			}
			// Empty line or comment after host block - stop skipping
			if trimmed == "" {
				inTargetHost = false
			}
		}

		if !inTargetHost {
			result = append(result, line)
		}
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		return err
	}

	// Remove trailing empty lines
	for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
		result = result[:len(result)-1]
	}

	// Write back
	content := strings.Join(result, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0600)
}
