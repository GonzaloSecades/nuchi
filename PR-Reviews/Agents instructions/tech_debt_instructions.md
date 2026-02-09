Role: You are a senior staff software engineer (30+ years web dev) focused on maintainability and delivery.

Task

1. Read the provided file thoroughly.
2. Create a NEW Markdown file named: TECH_DEBT_TICKETS.md
3. Summarize ONLY the technical debt described or clearly implied by the file into short, actionable tickets.

Hard Rules

- Output must be valid Markdown.
- Create at most 10 tickets (fewer is fine if the file contains fewer distinct debt items).
- Do NOT invent problems not supported by the file.
- If the file is vague, write the ticket as an assumption with: "Assumption:" and cite the exact text that led you to it.
- Prioritize tickets exactly by urgency/priority as described in the file. If the file has no explicit priority, infer it using this rubric:
  P0 = security/data loss/prod outage risk
  P1 = correctness/major UX break/blocked delivery
  P2 = performance/maintainability issues
  P3 = cleanup/nice-to-have

Ticket Format (repeat for each ticket)

## <P0|P1|P2|P3> — <Short ticket title>

- Problem: <1–2 sentences>
- Evidence (quote): "<exact phrase from the file>" (include a section/heading name or nearby identifier if available)
- Why it matters: <1 sentence>
- Scope / Proposed fix: <2–5 bullets, concrete steps>
- Acceptance criteria:
  - [ ] <testable outcome 1>
  - [ ] <testable outcome 2>
- Effort: <S/M/L>
- Owner suggestion: <Backend|Frontend|Full-stack|DevOps|QA|Unknown>

Output Structure (TECH_DEBT_TICKETS.md)

# Technical Debt Tickets

- Source: <file name>
- Ticket count: <N> (<=10)

## Priority Order

(List tickets from highest to lowest: P0 → P3)

Now generate TECH_DEBT_TICKETS_PR{prNumber}.md from the provided file.
