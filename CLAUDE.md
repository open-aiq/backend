# Project Rules

Project knowledge is NOT duplicated here — it lives in the docs below.
Read the relevant one before implementing changes that touch project code.

## Read before implementing
- Architecture & domain conventions → README.md (§ Architecture)
- Database, code generation, migrations → README.md (§ Database)
- Available commands → `make help`
- Configuration variables → `.env.example`

## Behavioral rules (always apply)
- Do NOT run commands (build, test, make, etc.) unless asked.
  You may ask permission to run `make build` / `make generate` to verify a change.
- Do NOT commit or push unless explicitly asked.
- Do NOT create files unless necessary.
- Ask before making destructive changes.

## Code style
- Keep it simple; avoid over-engineering and premature abstractions.
