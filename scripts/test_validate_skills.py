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


if __name__ == "__main__":
    unittest.main()
