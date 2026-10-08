import importlib.util
import json
import unittest
from pathlib import Path

p = Path(__file__).resolve().parents[1] / "verify_release_receipt.py"
s = importlib.util.spec_from_file_location("hermes_receipt", p)
m = importlib.util.module_from_spec(s)
s.loader.exec_module(m)
DIGEST = "sha256:" + "c"*64


class VerifyReleaseReceiptTests(unittest.TestCase):
    def setUp(self):
        self.ref = {"ref": "refs/tags/v0.21.6", "object": {"type": "tag", "sha": "a"*40}}
        self.tag = {"sha": "a"*40, "tag": "v0.21.6",
                    "object": {"type": "commit", "sha": "b"*40},
                    "message": json.dumps({"schema": 1, "version": "0.21.6",
                                           "commit": "b"*40, "dockerManifestDigest": DIGEST})}

    def test_verified_official_digest(self):
        self.assertEqual(m.verify("v0.21.6", self.ref, self.tag), DIGEST)

    def test_unannotated_and_wrong_version_rejected(self):
        for name in ("latest", "v0.21.7", "main"):
            with self.assertRaises(ValueError):
                m.verify(name, self.ref, self.tag)
        self.ref["object"]["type"] = "commit"
        with self.assertRaisesRegex(ValueError, "annotated"):
            m.verify("v0.21.6", self.ref, self.tag)

    def test_bad_receipt_fails_closed(self):
        for k, bad in (("schema", 2), ("version", "0.22.0"), ("commit", "f"*40),
                       ("dockerManifestDigest", "sha256:invalid")):
            original = self.tag["message"]
            body = json.loads(original)
            body[k] = bad
            self.tag["message"] = json.dumps(body)
            with self.assertRaises(ValueError):
                m.verify("v0.21.6", self.ref, self.tag)
            self.tag["message"] = original

    def test_missing_receipt_and_tag_object_drift(self):
        self.tag["sha"] = "f"*40
        with self.assertRaises(ValueError):
            m.verify("v0.21.6", self.ref, self.tag)
        self.tag["sha"] = "a"*40
        self.tag["message"] = "not-json"
        with self.assertRaises(ValueError):
            m.verify("v0.21.6", self.ref, self.tag)


if __name__ == "__main__":
    unittest.main()
