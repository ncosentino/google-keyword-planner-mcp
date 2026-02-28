#!/usr/bin/env python3
# Placeholder -- full implementation in a later commit
import sys
output = sys.argv[1] if len(sys.argv) > 1 else "docs/articles.md"
with open(output, "w") as f:
    f.write("# Articles\n\nPlaceholder.\n")
