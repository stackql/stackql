# Skills served over MCP

Each directory here is one Agent Skill that the MCP server serves when
`server.extensions.skills.enabled` is set (see `docs/mcp.md`, "MCP extensions"). This README is for
maintainers: plain files at this level are not skills and are not served.

## Adding a skill

1. Create `<name>/SKILL.md` with YAML frontmatter between `---` lines. `name` must equal the
   directory name: 1 to 64 characters of `a-z`, `0-9` and `-`, with no leading, trailing or doubled
   hyphen. `description` is 1 to 1024 characters and says when to load the skill. Every other
   frontmatter field (`license`, `compatibility`, `metadata`, ...) is published unchanged.
2. Keep SKILL.md under 500 lines and roughly 5,000 tokens. Put detail in `references/` and worked
   examples in `assets/`, and point to them from SKILL.md by relative path.
3. Start every reference file condensed from another source with a line naming the repository,
   tag or commit, and path, plus its licence.
4. Use only these file types, which map to the MIME types served by `resources/read`: `.md`
   (`text/markdown`), `.yml` / `.yaml` (`application/yaml`) and `.iql` (`text/plain`). Add a new
   type to `skillMIMETypes` in `skills.go` before using it.
5. Stay within 512 files and 16 MiB per skill. Files are served as UTF-8 with LF line endings.

`TestEmbeddedSkillsAreValid` in `skills_test.go` loads every skill here and fails on a frontmatter,
naming, size or file type violation, so a bad skill cannot merge. The same checks run at server
startup when skills are enabled.
