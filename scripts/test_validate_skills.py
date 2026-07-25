from pathlib import Path
from tempfile import TemporaryDirectory
import unittest

from validate_skills import validate_skill


class ValidateSkillTest(unittest.TestCase):
    def make_skill(self, root: Path, name: str, description: str) -> Path:
        skill = root / name
        (skill / "agents").mkdir(parents=True)
        (skill / "SKILL.md").write_text(
            f"---\nname: {name}\ndescription: {description}\n---\n\n# Skill\n",
            encoding="utf-8",
        )
        (skill / "agents" / "openai.yaml").write_text(
            "interface:\n"
            f"  display_name: \"{name}\"\n"
            "  short_description: \"A useful project skill description\"\n"
            f"  default_prompt: \"Use ${name} for this task.\"\n",
            encoding="utf-8",
        )
        return skill

    def test_valid_skill_has_no_errors(self):
        with TemporaryDirectory() as tmp:
            skill = self.make_skill(
                Path(tmp),
                "implementing-locker-orders",
                "Use when changing locker order behavior",
            )
            self.assertEqual(validate_skill(skill), [])

    def test_description_must_start_with_use_when(self):
        with TemporaryDirectory() as tmp:
            skill = self.make_skill(
                Path(tmp), "implementing-locker-orders", "Build locker orders"
            )
            self.assertIn(
                "description must start with 'Use when'", validate_skill(skill)
            )

    def test_site_device_skill_captures_required_invariants(self):
        skill = Path(".agents/skills/developing-site-device-domain")
        self.assertEqual(validate_skill(skill), [])
        reference = skill / "references" / "invariants.md"
        self.assertTrue(reference.is_file(), "missing site/device invariants reference")
        text = (skill / "SKILL.md").read_text(encoding="utf-8")
        text += "\n" + reference.read_text(encoding="utf-8")
        required = [
            "MySQL is the source of truth",
            "Redis is never authoritative",
            "idempotency",
            "X-Internal-Token",
            "production",
            "simulator",
            "UTC",
            "redact",
        ]
        for phrase in required:
            with self.subTest(phrase=phrase):
                self.assertIn(phrase.lower(), text.lower())


if __name__ == "__main__":
    unittest.main()
