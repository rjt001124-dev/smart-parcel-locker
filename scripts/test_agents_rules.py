from pathlib import Path
import unittest


class AgentsRulesTest(unittest.TestCase):
    def test_required_rules_are_present(self):
        text = Path("AGENTS.md").read_text(encoding="utf-8")
        required = [
            "Go Kratos",
            "整数分",
            "订单状态机",
            "幂等",
            "Proto",
            "测试先行",
            "Figma",
            "人工确认",
        ]
        for phrase in required:
            with self.subTest(phrase=phrase):
                self.assertIn(phrase, text)


if __name__ == "__main__":
    unittest.main()
