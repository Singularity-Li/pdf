package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/giraffesyo/pdf/internal/cjk"
)

func TestGB1WholeOfficialSourceAndReproducibleData(t *testing.T) {
	path := "../../data/Adobe-GB1-UCS2"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "368b40ec05568faf323cfbbddf23c60d566bbc88e366981eb6541b026381a3b8" {
		t.Fatal("pinned official source changed")
	}
	table, err := readGB1(path)
	if err != nil || len(table) != 30284 {
		t.Fatalf("incomplete official table: %d %v", len(table), err)
	}
	scalars := map[uint32]rune{}
	sequences := map[uint32]string{}
	for cid, want := range table {
		got, ok := cjk.Unicode("GB1", cid)
		if !ok || got != want {
			t.Fatalf("CID %d = %q,%v; official %q", cid, got, ok, want)
		}
		runes := []rune(want)
		if len(runes) == 1 {
			scalars[cid] = runes[0]
		} else {
			sequences[cid] = want
		}
	}
	if len(sequences) != 4 {
		t.Fatal("lost variation sequences")
	}
	binary, err := encode(scalars)
	if err != nil {
		t.Fatal(err)
	}
	jsonData, _ := json.Marshal(sequences)
	for name, want := range map[string][]byte{"GB1.bin": binary, "GB1_sequences.json": append(jsonData, '\n')} {
		got, err := os.ReadFile(filepath.Join("../../data", name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("generated %s is not reproducible: %v", name, err)
		}
	}
}

func TestGB1InvalidSourceFails(t *testing.T) {
	for _, text := range []string{
		"2 beginbfchar\n<0001> <0031>\nendbfchar\n",
		"1 beginbfchar\n<0001> <0031>\n",
		"2 beginbfchar\n<0001> <0031>\n<0001> <0032>\nendbfchar\n",
		"1 beginbfchar\n<0001> <D800>\nendbfchar\n",
		"1 beginbfrange\n<0002> <0001> <0031>\nendbfrange\n",
	} {
		path := filepath.Join(t.TempDir(), "bad-cmap")
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readGB1(path); err == nil {
			t.Fatalf("invalid source accepted: %q", text)
		}
	}
}
