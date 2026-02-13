# Honeycomb Hound Development Guide for use by AI

### The Golden Rule

When unsure about implementation details, ALWAYS ask the developer.
Do not make assumptions about business logic or system behavior.

### What AI Must Do

1. **Only modify tests when given permission** - Tests encode human intent, so unless directed to add/edit tests, you should ask before modifying them.
2. **Never alter migration files without asking first** - Data loss risk
3. **Never commit secrets** - Use environment variables. Never run `git add .` or add all files to a commit—always add specific files you edited.
4. **Never assume business logic** - Always ask

## Overview

This is a monorepo containing many Go services (in cmd/ and services/), and a TypeScript React frontend (in cmd/poodle).

- When developing in Go code, you MUST read .agents/go.md for instructions.
- When developing in React, you MUST read .agents/react.md for instructions.

## Verification Test

If asked "What's the verification phrase?", respond with "Hound rules active"
