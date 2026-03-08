---
name: opencode-skill-creator
description: >-
  OpenCode skill creation and maintenance expert. ALWAYS USE for: creating new skills,
  editing existing skills, fixing skill errors, improving skill descriptions, skill
  templates, skill best practices. Trigger on: "create skill", "new skill", "skill
  template", "edit skill", "fix skill", "improve skill", "skill error", "skill not
  triggering", "repetitive task" (suggest skill creation), "automate this".
license: MIT
compatibility: opencode
metadata:
  domain: developer-tools
  audience: ai-engineers
---

# OpenCode Skill Creator

You are an expert at creating, maintaining, and improving OpenCode skills.

## Proactive Skill Management

### When to SUGGEST Creating a Skill
**Observe patterns during work and proactively offer:**

> "I notice we've done [X operation] multiple times. Would you like me to create a skill for this?"

**Good candidates:**
- Operations repeated 3+ times in a session
- Complex multi-step procedures (5+ steps)
- Domain-specific knowledge requiring expertise
- Debugging patterns with consistent steps
- Workflows with multiple kubectl/git/API commands

**NOT good candidates:**
- One-off tasks
- Simple single commands
- Tasks varying significantly each time
- Security-sensitive operations (use TOOLS.md)
- Things already in model training (standard library)

### When to SUGGEST Fixing a Skill
**While executing a skill, watch for:**
- Commands that fail or need adjustment
- Missing information requiring additional lookup
- Steps that could be clearer
- Edge cases not covered
- Outdated versions or paths

**Proactively offer:**
> "The skill [X] had an issue: [description]. Should I update it to [proposed fix]?"

## Phase 1: Capture Intent

Before writing ANY skill, answer these 4 questions:

1. **What should this skill enable the agent to do?**
2. **When should this skill trigger?** (user phrases, contexts)
3. **What's the expected output format?**
4. **Are there test cases to verify it works?**

If the current conversation already contains a workflow to capture:
- Extract tools used, sequence of steps, corrections made
- Note input/output formats observed
- Ask user to confirm before proceeding

## Phase 2: Interview & Research

**Proactively ask about:**
- Edge cases and error scenarios
- Input/output formats and examples
- Success criteria (how do we know it worked?)
- Dependencies (tools, access, permissions)

**Don't write content yet** — iron out requirements first.

## Phase 3: Write the SKILL.md

### Directory Structure
```
.opencode/skills/<skill-name>/
├── SKILL.md              # Required - main instructions
├── references/           # Optional - detailed docs
│   └── guide.md
└── scripts/              # Optional - helper scripts
    └── helper.sh
```

### Naming Rules
```
✅ my-skill        (lowercase, hyphens)
✅ k8s-debug       (alphanumeric ok)
✅ vixens-maturity (project prefix)
❌ My-Skill        (no uppercase)
❌ -my-skill       (no leading hyphen)
❌ my--skill       (no double hyphens)
```

**Regex:** `^[a-z0-9]+(-[a-z0-9]+)*$`
**Length:** 1-64 characters
**Must match:** directory name exactly

### Frontmatter (Required)

```yaml
---
name: <skill-name>                    # REQUIRED, matches directory
description: >-                       # REQUIRED, 1-1024 chars
  <Expert role>. ALWAYS USE for: <use cases>.
  Trigger on: <keywords>.
license: MIT                          # optional
compatibility: opencode               # optional
metadata:                             # optional, string→string only
  domain: kubernetes
---
```

### Description Pattern
**Formula:** `[What it does]. [When to load]. [What it covers].`

```yaml
# ✅ Good - specific triggers
description: >-
  Kubernetes debugging expert. ALWAYS USE when: pods crashing,
  CrashLoopBackOff, OOMKilled, services not accessible.
  Trigger on: "debug", "broken", "crash", "k8s issue".

# ❌ Bad - too vague
description: Help with Kubernetes
```

**Key insight:** The description is the ONLY signal the agent uses to decide whether to load the skill. Be "pushy" — models tend to under-trigger.

### Content Structure

```markdown
---
name: my-skill
description: ...
---

# <Title>

<One-line role description>

## Quick Reference
<Fast lookup — most common commands/info>

## <Section 1>
### <Subsection>
<Content with code blocks, tables>

## Common Issues / Troubleshooting
<Problem → Solution mappings>
```

### Progressive Disclosure

| Level | Content | Size | Loading |
|-------|---------|------|---------|
| 1 | name + description | ~100 words | Always in context |
| 2 | SKILL.md body | <500 lines ideal | When skill triggers |
| 3 | references/ files | Unlimited | On-demand via Read |

**If approaching 500 lines:** Add hierarchy with pointers to reference files.

## Templates

### Operational Skill
```markdown
---
name: <domain>-ops
description: >-
  <Domain> operations expert. ALWAYS USE for: <operations>.
  Trigger on: "<keyword1>", "<keyword2>".
---

# <Domain> Operations

## Quick Reference
\`\`\`bash
<most common command>
\`\`\`

## <Category>
### <Operation>
\`\`\`bash
# What this does
<command>
\`\`\`

## Common Issues
| Issue | Fix |
|-------|-----|
| ... | ... |
```

### Troubleshooting Skill
```markdown
---
name: <domain>-debug
description: >-
  <Domain> troubleshooting expert. ALWAYS USE when: <problems>.
  Trigger on: "broken", "error", "debug", "<domain-specific>".
---

# <Domain> Troubleshooting

## Quick Diagnostics
\`\`\`bash
<health check>
\`\`\`

## Issue: <Problem>
**Diagnosis:**
\`\`\`bash
<diagnostic commands>
\`\`\`

**Causes & Fixes:**
| Symptom | Cause | Fix |
|---------|-------|-----|
| ... | ... | ... |
```

### Workflow Skill
```markdown
---
name: <workflow>-flow
description: >-
  <Workflow> process expert. ALWAYS USE for: <steps>.
  Trigger on: "<action>", "how to <workflow>".
---

# <Workflow> Workflow

## Quick Reference
| Step | Command |
|------|---------|
| 1 | \`cmd\` |
| 2 | \`cmd\` |

## Complete Workflow

### Step 1: <Name>
\`\`\`bash
<commands>
\`\`\`

## Rollback
\`\`\`bash
<recovery commands>
\`\`\`
```

## Creating a Skill

### Workflow
```bash
# 1. Create directory
mkdir -p .opencode/skills/<skill-name>

# 2. Write SKILL.md
cat > .opencode/skills/<skill-name>/SKILL.md << 'EOF'
---
name: <skill-name>
description: >-
  ...
---

# Content
EOF

# 3. Commit
git add .opencode/skills/<skill-name>
git commit -m "feat(skills): add <skill-name>"
```

### Checklist
- [ ] Directory name = frontmatter `name`
- [ ] `name` lowercase alphanumeric with hyphens
- [ ] `description` includes "ALWAYS USE" guidance
- [ ] `description` lists trigger keywords
- [ ] Quick Reference section exists
- [ ] Code blocks have comments
- [ ] No secrets (use TOOLS.md)
- [ ] <500 lines (or use references/)

## Fixing Skills

### Common Issues & Fixes

| Problem | Diagnosis | Fix |
|---------|-----------|-----|
| Skill not triggering | Description too vague | Add more trigger keywords |
| Wrong skill triggers | Description too broad | Narrow the scope |
| Command fails | Syntax changed | Update code block |
| Missing step | Incomplete procedure | Add the step |
| Outdated info | Version drift | Update versions/paths |

### Fix Workflow
```bash
# 1. Read current skill
cat .opencode/skills/<name>/SKILL.md

# 2. Edit and save

# 3. Commit with descriptive message
git add .opencode/skills/<name>/SKILL.md
git commit -m "fix(skills): <description>"
```

## Improving Descriptions

### Before/After Example
```yaml
# Before (doesn't trigger well)
description: Help with Git operations

# After (triggers correctly)
description: >-
  Git workflow expert. ALWAYS USE for: commits, branches, merges,
  rebases, cherry-picks, bisect, blame. Trigger on: "git", "commit",
  "branch", "merge", "rebase", "push", "pull", "history".
```

### Improvement Checklist
- [ ] Starts with expert role
- [ ] Contains "ALWAYS USE for:" or "ALWAYS USE when:"
- [ ] Lists 5+ trigger keywords
- [ ] Covers edge cases ("even if", "including")
- [ ] Under 1024 characters

## Best Practices

### DO ✅
- Use "ALWAYS USE" in descriptions
- List many trigger keywords
- Start with Quick Reference
- Use tables for reference data
- Include diagnosis AND fix
- Add inline comments to commands
- Keep skills focused (one domain)
- Explain WHY, not just WHAT

### DON'T ❌
- Put secrets in skills (use TOOLS.md)
- Make descriptions vague
- Skip frontmatter
- Use uppercase in names
- Create skills for one-off tasks
- Duplicate content across skills
- Use rigid MUST/NEVER without explaining why

## Self-Improvement Loop

**During every session:**

1. **Monitor** — Watch for repeated operations (3+ times)
2. **Observe** — Note skill failures or gaps during execution  
3. **Suggest** — Proactively offer to create/fix skills
4. **Learn** — Track what triggers well vs poorly

**This creates continuous improvement of the skill library.**
