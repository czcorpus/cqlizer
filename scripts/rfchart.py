import csv
import sys

import matplotlib.pyplot as plt

# Read arguments: script_name, "-o", output_path, "-t", title
if len(sys.argv) < 5 or sys.argv[1] != "-o" or sys.argv[3] != "-t":
    print("Error: Expected arguments: -o <output_path> -t <title>", file=sys.stderr)
    sys.exit(1)

output_file = sys.argv[2]
title = sys.argv[4]

# Read CSV from stdin
reader = csv.reader(sys.stdin, delimiter=";")
headers = next(reader)
rows = list(reader)

if not rows:
    print("Error: CSV contains no data rows", file=sys.stderr)
    sys.exit(1)

x_column = headers[0]
y_columns = headers[1:]

# Convert data
x_data = [float(row[0]) for row in rows]
y_datasets = [
    [float(row[col_idx]) for row in rows] for col_idx in range(1, len(headers))
]

# Create plot
plt.figure(figsize=(10, 6))
plt.ylim(0, 1)
markers = ["o", "s", "^", "D", "v", "<", ">", "p", "*", "h"]

for idx, (y_data, col_name) in enumerate(zip(y_datasets, y_columns)):
    marker = markers[idx % len(markers)]
    plt.plot(x_data, y_data, marker=marker, label=col_name, linewidth=2, markersize=8)

plt.xlabel(x_column, fontsize=12)
plt.ylabel("Values", fontsize=12)
plt.title(title, fontsize=14)
plt.legend(fontsize=10, loc="lower left")
plt.grid(True, alpha=0.3)
plt.tight_layout()
plt.savefig(output_file, dpi=300, bbox_inches="tight")
