# FAQ — Design Decisions & Deployment

## Skill Template: Inline vs. Separate Files

**Decision: Keep templates inline in SKILL.md (current stage)**

| Approach | Pros | Cons |
|----------|------|------|
| Inline in SKILL.md | Single file to load, zero extra I/O, simpler maintenance | File grows with more templates |
| Separate `templates/` folder | Clean separation, easier to add new templates | Requires extra `Read` calls at runtime, more failure points |

**Rationale:**

- Claude Code loads only `SKILL.md` when a skill is invoked. Any additional files require explicit `Read` tool calls, adding latency and potential failure points.
- The current Universal Template is ~20 lines — not large enough to justify separation.
- One skill = one file keeps the mental model simple for contributors.

**When to split:** When the project supports multiple template types (e.g., Custom Node, Post Process, Decal, Particle), each with distinct formats. At that point, the recommended structure would be:

```
skills/uetx-material/
├── SKILL.md                  ← Main logic + routing
├── templates/
│   ├── custom-node.hlsl      ← Custom Node template
│   ├── post-process.hlsl     ← Post Process template
│   └── decal.hlsl            ← Decal template
```

---

## Windows Deployment: Project-local vs. Global PATH

**Decision: Global PATH (recommended)**

| Approach | Install | Multi-version | Skill Compatibility | Target User |
|----------|---------|---------------|---------------------|-------------|
| Project-local binary | Zero-config, unzip and run | Natural (each project has its own) | Requires absolute/relative paths | Artists/TAs |
| Global PATH | Need to set PATH once | Manual switching | Works out of the box (`uetx generate`) | Developers/CI |

**Rationale:**

1. **Skill execution relies on PATH** — Claude Code's Bash tool looks up commands via the system PATH. Hardcoding paths in the skill is not portable across machines.
2. **`go install` puts the binary in `$GOPATH/bin`** — which is typically already in PATH for Go developers. Zero extra setup.
3. **Release binaries** (from GitHub Releases) can be placed in any PATH directory:
   - macOS/Linux: `/usr/local/bin/` or `~/bin/`
   - Windows: `C:\Users\<user>\bin\` (add to PATH once)
4. **The skill includes a pre-check** that verifies `uetx` is available before attempting conversion, with a clear error message if not found.

**Windows-specific setup:**

```powershell
# Option A: go install (recommended if Go is installed)
go install github.com/radial/uetx/cmd/uetx@latest

# Option B: Download release binary
# 1. Download uetx_x.x.x_windows_amd64.zip from GitHub Releases
# 2. Extract uetx.exe to a directory in your PATH, e.g.:
mkdir -Force "$env:USERPROFILE\bin"
Move-Item uetx.exe "$env:USERPROFILE\bin\"
# 3. Add to PATH (one-time):
[Environment]::SetEnvironmentVariable("Path", "$env:Path;$env:USERPROFILE\bin", "User")
```

**macOS/Linux setup:**

```bash
# Option A: go install
go install github.com/radial/uetx/cmd/uetx@latest

# Option B: Download release binary
# Extract and move to PATH
tar xzf uetx_x.x.x_darwin_arm64.tar.gz
mv uetx /usr/local/bin/
```
