import sys
import os
import itertools

# Path to ../liborchest-python
parent_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
# print(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, parent_path)

import liborchest_python as OLib


data = []

while True:
    d = OLib.read_stdin_bytes()
    if d == b"":
        break
    data.append(d.decode('UTF-8'))

string = str("".join(list(itertools.chain(*data))))

OLib.write_stdout_string(string.replace(" ", "\\ "))
