package mcp_server //nolint:testpackage,revive // exercise internal wiring

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

// Every embedded skill must load: a frontmatter, naming, size or MIME type
// violation fails here, so a bad skill cannot merge.
func TestEmbeddedSkillsAreValid(t *testing.T) {
	skills, err := loadEmbeddedSkills()
	if err != nil {
		t.Fatalf("loadEmbeddedSkills: %v", err)
	}
	names := map[string]embeddedSkill{}
	for _, s := range skills {
		names[s.Name()] = s
	}
	for _, want := range []string{"stackql-deploy", "stackql-iac"} {
		if _, ok := names[want]; !ok {
			t.Errorf("embedded skill %q is missing (have %v)", want, keysOf(names))
		}
	}
	for _, s := range skills {
		for _, f := range s.Files() {
			checkServedSkillFile(t, s, f)
		}
	}
}

func checkServedSkillFile(t *testing.T, s embeddedSkill, f skillFile) {
	t.Helper()
	sum := sha256.Sum256([]byte(f.Text()))
	if f.Digest() != "sha256:"+hex.EncodeToString(sum[:]) || f.Size() != int64(len(f.Text())) {
		t.Errorf("%s: digest or size does not describe the served bytes", f.URI())
	}
	if strings.Contains(f.Text(), "\r\n") {
		t.Errorf("%s: served bytes contain CRLF", f.URI())
	}
	if lines := strings.Count(f.Text(), "\n"); f.URI() == s.URI() && lines >= 500 {
		t.Errorf("%s: %d lines, keep SKILL.md under 500", f.URI(), lines)
	}
}

func keysOf(m map[string]embeddedSkill) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func skillFS(dir, frontmatter string, extra map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{
		"skills/README.md":                     {Data: []byte("maintainer notes, not served\n")},
		"skills/" + dir + "/" + skillEntryFile: {Data: []byte("---\n" + frontmatter + "\n---\n\nBody.\n")},
	}
	for name, body := range extra {
		fsys["skills/"+dir+"/"+name] = &fstest.MapFile{Data: []byte(body)}
	}
	return fsys
}

func TestSkillLoaderBuildsCompleteManifest(t *testing.T) {
	fsys := skillFS("demo", "name: demo\ndescription: A demo skill.\nmetadata:\n  owner: robot\n",
		map[string]string{"references/a.md": "a\r\nb\r\n", "assets/x/stack.yml": "k: v\n", "assets/x/r.iql": "select 1;\n"})
	skills, err := loadSkillsFrom(fsys, "skills")
	if err != nil {
		t.Fatalf("loadSkillsFrom: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("loaded %d skills, want 1 (README.md is not a skill)", len(skills))
	}
	s := skills[0]
	if s.URI() != "skill://demo/SKILL.md" {
		t.Fatalf("uri = %s", s.URI())
	}
	if owner := s.Frontmatter()["metadata"].(map[string]any)["owner"]; owner != "robot" {
		t.Fatalf("frontmatter not passed through: %v", s.Frontmatter())
	}
	got := map[string]string{}
	for _, f := range s.Files() {
		got[f.URI()] = f.MIMEType()
	}
	want := map[string]string{
		"skill://demo/SKILL.md":           "text/markdown",
		"skill://demo/references/a.md":    "text/markdown",
		"skill://demo/assets/x/stack.yml": "application/yaml",
		"skill://demo/assets/x/r.iql":     "text/plain",
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("manifest = %v, want %v", got, want)
	}
	for _, f := range s.Files() {
		if f.URI() == "skill://demo/references/a.md" && (f.Text() != "a\nb\n" || f.Size() != 4) {
			t.Fatalf("CRLF not normalised before digesting: %q", f.Text())
		}
	}
}

func TestSkillLoaderRejectsInvalidSkills(t *testing.T) {
	long := strings.Repeat("x", maxSkillDescriptionLength+1)
	many := map[string]string{}
	for i := 0; i < maxSkillFiles; i++ {
		many[fmt.Sprintf("references/%03d.md", i)] = "x\n"
	}
	cases := []struct {
		name string
		fsys fstest.MapFS
		want string
	}{
		{"uppercase name", skillFS("Demo", "name: Demo\ndescription: d", nil), "must be 1 to 64"},
		{"doubled hyphen", skillFS("de--mo", "name: de--mo\ndescription: d", nil), "must be 1 to 64"},
		{"leading hyphen", skillFS("-demo", "name: -demo\ndescription: d", nil), "must be 1 to 64"},
		{"trailing hyphen", skillFS("demo-", "name: demo-\ndescription: d", nil), "must be 1 to 64"},
		{"name too long", skillFS(strings.Repeat("a", 65), "name: "+strings.Repeat("a", 65)+"\ndescription: d", nil),
			"must be 1 to 64"},
		{"name mismatch", skillFS("demo", "name: other\ndescription: d", nil), "must match its directory"},
		{"missing description", skillFS("demo", "name: demo", nil), "description must be"},
		{"description too long", skillFS("demo", "name: demo\ndescription: "+long, nil), "description must be"},
		{"too many files", skillFS("demo", "name: demo\ndescription: d", many), "file limit"},
		{"too many bytes", skillFS("demo", "name: demo\ndescription: d",
			map[string]string{"references/big.md": strings.Repeat("x", maxSkillBytes)}), "byte limit"},
		{"unknown extension", skillFS("demo", "name: demo\ndescription: d", map[string]string{"run.sh": "x"}),
			"no known MIME type"},
		{"bad frontmatter", skillFS("demo", "name: [demo", nil), "frontmatter"},
		{"no frontmatter", fstest.MapFS{"skills/demo/SKILL.md": {Data: []byte("Body only.\n")}}, "frontmatter"},
		{"no SKILL.md", fstest.MapFS{"skills/demo/references/a.md": {Data: []byte("x\n")}}, "missing SKILL.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadSkillsFrom(tc.fsys, "skills")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestSkillsCacheFieldsOnlyOnSessionlessRevision(t *testing.T) {
	if got := marshalString(t, newSkillsCacheFields("2025-11-25")); got != `{}` {
		t.Fatalf("legacy cache fields = %s, want none", got)
	}
	if got := marshalString(t, newSkillsCacheFields(protocolVersionSessionless)); got !=
		`{"resultType":"complete","ttlMs":30000,"cacheScope":"private"}` {
		t.Fatalf("2026-07-28 cache fields = %s", got)
	}
}
