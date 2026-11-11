"""Policy compilation regression tests; run with python3 -m unittest discover."""
import copy
import json
import unittest
from generate_seccomp import ROOT, ARCHITECTURES, generate


class ProfileTests(unittest.TestCase):
    def setUp(self):
        self.source = json.loads((ROOT / 'seccomp.json').read_text())

    def test_committed_profiles(self):
        for arch in ARCHITECTURES:
            expected = json.dumps(generate(self.source, arch), indent=2) + '\n'
            self.assertEqual(expected, (ROOT / ('seccomp-' + arch + '.json')).read_text())

    def test_unknown_policy_metadata_fails(self):
        for field in ('includes', 'excludes'):
            source = copy.deepcopy(self.source)
            source['syscalls'][0][field] = {'futureFilter': ['required']}
            with self.assertRaises(ValueError):
                generate(source, 'arm64')

    def test_future_kernel_gate_fails(self):
        self.source['syscalls'][0]['includes'] = {'minKernel': '99.1'}
        with self.assertRaises(ValueError):
            generate(self.source, 'amd64')

    def test_argument_filters_survive(self):
        rule = {'names': ['example'], 'action': 'SCMP_ACT_ALLOW',
                'args': [{'index': 0, 'value': 7, 'op': 'SCMP_CMP_EQ'}],
                'includes': {}, 'excludes': {'caps': ['CAP_SYS_ADMIN']}}
        self.source['syscalls'] = [rule]
        result = generate(self.source, 'amd64')['syscalls']
        self.assertEqual(result, [{k: rule[k] for k in ('names', 'action', 'args')}])


if __name__ == '__main__':
    unittest.main()
