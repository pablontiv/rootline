#!/usr/bin/env bash
# Validate conventional commit format on the first line of the commit message.
# Pattern: type(scope)?: description
# Valid types: feat, fix, chore, docs, refactor, perf, test, style, ci

commit_msg_file="$1"
first_line=$(head -n1 "$commit_msg_file")

pattern='^(feat|fix|chore|docs|refactor|perf|test|style|ci)(\(.+\))?!?: .+'

if ! [[ "$first_line" =~ $pattern ]]; then
  echo "ERROR: Commit message does not follow conventional commits format."
  echo ""
  echo "Expected: <type>(<scope>): <description>"
  echo ""
  echo "Valid types: feat, fix, chore, docs, refactor, perf, test, style, ci"
  echo ""
  echo "Examples:"
  echo "  feat: add user authentication"
  echo "  fix(index): correct traversal order"
  echo "  feat!: breaking change in CLI"
  echo ""
  echo "Your message: $first_line"
  exit 1
fi
