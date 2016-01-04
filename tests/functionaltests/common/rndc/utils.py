import subprocess
import re


def run_cmd(*cmd):
    p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    out, err = p.communicate()
    if p.returncode != 0:
        raise Exception("cmd `{0}` failed:\n\t{1}".format(" ".join(cmd), err))
    return (out.decode('utf-8'), err.decode('utf-8'), p.returncode)


def parse_rndc_version(out):
    """Return a tuple like (9, 9, 5)

    :param out: a string containing the output of `rndc status`
    """
    version_line = out.split('\n')[0]
    assert version_line.startswith('version')
    parts = re.findall("\w+[.]\w+[.]\w+", version_line)[0].split('.')
    return tuple(map(int, parts))
