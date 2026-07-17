from pathlib import Path
import re
import sys


NAME_RE = re.compile(r"^[a-z0-9-]{1,64}$")


def parse_frontmatter(text: str) -> dict[str, str]:
    if not text.startswith("---\n"):
        return {}
    end = text.find("\n---\n", 4)
    if end == -1:
        return {}
    fields: dict[str, str] = {}
    for line in text[4:end].splitlines():
        if ":" in line:
            key, value = line.split(":", 1)
            fields[key.strip()] = value.strip().strip('"')
    return fields


def validate_skill(skill: Path) -> list[str]:
    errors: list[str] = []
    skill_md = skill / "SKILL.md"
    openai_yaml = skill / "agents" / "openai.yaml"
    if not skill_md.is_file():
        return ["missing SKILL.md"]

    fields = parse_frontmatter(skill_md.read_text(encoding="utf-8"))
    name = fields.get("name", "")
    description = fields.get("description", "")
    if not NAME_RE.fullmatch(name):
        errors.append("invalid skill name")
    if name != skill.name:
        errors.append("skill name must match folder name")
    if not description.startswith("Use when"):
        errors.append("description must start with 'Use when'")
    if not openai_yaml.is_file():
        errors.append("missing agents/openai.yaml")
    elif f"${name}" not in openai_yaml.read_text(encoding="utf-8"):
        errors.append("default_prompt must mention the skill with $name")
    return errors


def main() -> int:
    root = Path(".agents/skills")
    failures = 0
    for skill in sorted(path for path in root.iterdir() if path.is_dir()):
        errors = validate_skill(skill)
        if errors:
            failures += 1
            print(f"{skill}: {', '.join(errors)}")
    if failures:
        return 1
    print("all project skills are valid")
    return 0


if __name__ == "__main__":
    sys.exit(main())
