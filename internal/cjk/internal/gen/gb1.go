package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Use Adobe's PDF CID-to-Unicode table rather than inverting Unicode-to-CID:
// the latter omits alternate glyph CIDs and can choose compatibility forms.
func generateGB1() error {
	table, err := readGB1("internal/cjk/data/Adobe-GB1-UCS2")
	if err != nil {
		return err
	}
	scalars := map[uint32]rune{}
	sequences := map[uint32]string{}
	for cid, text := range table {
		runes := []rune(text)
		if len(runes) == 1 {
			scalars[cid] = runes[0]
		} else {
			sequences[cid] = text
		}
	}
	binary, err := encode(scalars)
	if err != nil {
		return err
	}
	if err := os.WriteFile("internal/cjk/data/GB1.bin", binary, 0644); err != nil {
		return err
	}
	raw, err := json.Marshal(sequences)
	if err != nil {
		return err
	}
	return os.WriteFile("internal/cjk/data/GB1_sequences.json", append(raw, '\n'), 0644)
}

func readGB1(path string) (map[uint32]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	table := map[uint32]string{}
	section := ""
	expected, entries := 0, 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "%") {
			continue
		}
		last := fields[len(fields)-1]
		if last == "beginbfchar" || last == "beginbfrange" {
			if section != "" || len(fields) != 2 {
				return nil, fmt.Errorf("invalid GB1 mapping section")
			}
			expected, err = strconv.Atoi(fields[0])
			if err != nil || expected < 1 {
				return nil, fmt.Errorf("invalid GB1 mapping count")
			}
			entries = 0
			section = last
			continue
		}
		if last == "endbfchar" || last == "endbfrange" {
			if last != strings.Replace(section, "begin", "end", 1) || entries != expected {
				return nil, fmt.Errorf("incomplete GB1 mapping section")
			}
			section = ""
			continue
		}
		if section == "" {
			continue
		}
		count := 2
		if section == "beginbfrange" {
			count = 3
		}
		if len(fields) != count {
			return nil, fmt.Errorf("unsupported GB1 mapping line %q", scanner.Text())
		}
		entries++
		lo, err := strconv.ParseUint(strings.Trim(fields[0], "<>"), 16, 16)
		if err != nil {
			return nil, err
		}
		hi := lo
		if count == 3 {
			hi, err = strconv.ParseUint(strings.Trim(fields[1], "<>"), 16, 16)
			if err != nil || hi < lo {
				return nil, fmt.Errorf("invalid GB1 CID range")
			}
		}
		dst := strings.Trim(fields[count-1], "<>")
		base, err := strconv.ParseUint(dst, 16, 64)
		if err != nil || len(dst) == 0 || len(dst)%4 != 0 {
			return nil, fmt.Errorf("invalid GB1 Unicode value")
		}
		for cid := lo; cid <= hi; cid++ {
			raw, err := hex.DecodeString(fmt.Sprintf("%0*x", len(dst), base+cid-lo))
			if err != nil || len(raw)%2 != 0 {
				return nil, fmt.Errorf("invalid GB1 Unicode range")
			}
			units := make([]uint16, len(raw)/2)
			for i := range units {
				units[i] = uint16(raw[2*i])<<8 | uint16(raw[2*i+1])
			}
			text := string(utf16.Decode(units))
			if text == "" || !slices.Equal(units, utf16.Encode([]rune(text))) {
				return nil, fmt.Errorf("invalid GB1 UTF16")
			}
			if _, exists := table[uint32(cid)]; exists {
				return nil, fmt.Errorf("duplicate GB1 CID %d", cid)
			}
			table[uint32(cid)] = text
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if section != "" || len(table) == 0 {
		return nil, fmt.Errorf("incomplete GB1 table")
	}
	return table, nil
}
